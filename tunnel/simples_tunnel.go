package tunnel

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type SimplesTunnel struct {
	ApiClaim           api.Api
	ControlAddr        netip.AddrPort
	ControlChannel     *AuthenticatedControl
	UdpTunnel          UdpTunnel
	LastKeepAlive      uint64
	LastPing           uint64
	LastPong           uint64
	LastUdpAuth        uint64
	lastControlTargets []netip.AddrPort
}

func ControlAddresses(Api api.Api) ([]netip.AddrPort, error) {
	controls, err := Api.AgentRoutings(nil)
	if err != nil {
		return nil, err
	}

	addrs := []netip.AddrPort{}
	for _, v := range append(controls.Targets6, controls.Targets4...) {
		addrs = append(addrs, netip.AddrPortFrom(v, 5525))
	}
	return addrs, nil
}

func (Tun *SimplesTunnel) Setup() error {
	if err := AssignUdpTunnel(&Tun.UdpTunnel); err != nil {
		return err
	}

	addresses, err := ControlAddresses(Tun.ApiClaim)
	if err != nil {
		return err
	}

	setup, err := (&SetupFindSuitableChannel{Address: addresses}).Setup()
	if err != nil {
		return err
	}

	control_channel, err := setup.Authenticate(Tun.ApiClaim)
	if err != nil {
		return err
	}
	Tun.ControlAddr = setup.ControlAddr
	Tun.ControlChannel = control_channel
	return nil
}

func (Tun *SimplesTunnel) ReloadControlAddr() (bool, error) {
	addresses, err := ControlAddresses(Tun.ApiClaim)
	if err != nil {
		return false, err
	}

	if slices.ContainsFunc(Tun.lastControlTargets, func(a netip.AddrPort) bool {
		return !slices.ContainsFunc(addresses, func(b netip.AddrPort) bool {
			return a.Compare(b) == 0
		})
	}) {
		return false, nil
	}
	setup, err := (&SetupFindSuitableChannel{addresses}).Setup()
	if err != nil {
		return false, err
	}
	updated, err := Tun.UpdateControlAddr(*setup)
	Tun.lastControlTargets = addresses
	return updated, err
}

func (Tun *SimplesTunnel) UpdateControlAddr(conncted ConnectedControl) (ok bool, err error) {
	if conncted.ControlAddr.Compare(Tun.ControlAddr) == 0 {
		LogDebug.Println("not required Update control addr")
		return
	}

	var controlChannel *AuthenticatedControl
	controlChannel, err = conncted.Authenticate(Tun.ApiClaim)
	if err != nil {
		return
	}
	LogDebug.Printf("Update control address %s to %s\n", Tun.ControlAddr.String(), conncted.ControlAddr.String())

	Tun.ControlChannel = controlChannel
	Tun.ControlAddr = conncted.ControlAddr
	Tun.LastPing = 0
	Tun.LastKeepAlive = 0
	Tun.LastUdpAuth = 0
	Tun.UdpTunnel.InvalidateSession()
	ok = true
	return
}

func (Tun *SimplesTunnel) Update() (*NewClient, error) {
	if Tun.ControlChannel.IsIspired() {
		LogDebug.Println("Creating new controller channel...")
		if err := Tun.ControlChannel.Authenticate(); err != nil {
			LogDebug.Println(err)
			time.Sleep(time.Second * 2)
			return nil, nil
		}
	}

	now := uint64(time.Now().UnixMilli())
	if now-Tun.LastPing > 1_000 {
		Tun.LastPing = now
		if err := Tun.ControlChannel.SendPing(200, time.UnixMilli(int64(now))); err != nil {
			LogDebug.Println("failed to send ping")
		}
	}

	// d, _ := json.MarshalIndent(Tun, "", "  ")
	// LogDebug.Panicf(string(d))

	if Tun.UdpTunnel.RequiresAuth() {
		if 5_000 < now-Tun.LastUdpAuth {
			Tun.LastUdpAuth = now
			if err := Tun.ControlChannel.SendSetupUDPChannel(9000); err != nil {
				LogDebug.Println("failed to send udp setup request to control")
				LogDebug.Println(err)
			}
		}
	} else if Tun.UdpTunnel.RequireResend() {
		if 1_000 < now-Tun.LastUdpAuth {
			Tun.LastUdpAuth = now
			if _, err := Tun.UdpTunnel.ResendToken(); err != nil {
				LogDebug.Println("failed to send udp auth request")
				LogDebug.Println(err)
			}
		}

		timeTillExpire := func(x, y uint64) uint64 {
			if x > y {
				return y
			}
			return x
		}(uint64(Tun.ControlChannel.Registered.ExpiresAt.UnixMilli()), uint64(now))
		if 10_000 < now-Tun.LastKeepAlive && timeTillExpire < 30_000 {
			Tun.LastKeepAlive = now
			LogDebug.Println("send KeepAlive")
			if err := Tun.ControlChannel.SendKeepAlive(100); err != nil {
				LogDebug.Println("failed to send KeepAlive")
				LogDebug.Println(err)
			}
			if err := Tun.ControlChannel.SendSetupUDPChannel(1); err != nil {
				LogDebug.Println("failed to send setup udp channel request")
				LogDebug.Println(err)
			}
		}

		timeout := 0
		for range 30 {
			if timeout >= 10 {
				LogDebug.Println("feed recv timeout")
				break
			}
			LogDebug.Println("RX Feed message")
			men, err := Tun.ControlChannel.RecFeedMsg()
			if err != nil {
				timeout++
				LogDebug.Printf("failed to parse response: %s\n", err.Error())
				continue
			}

			if men.NewClient != nil {
				return men.NewClient, nil
			} else if men.Response != nil {
				cont := men.Response.Content
				if cont.UdpChannelDetails != nil {
					LogDebug.Print("Response SetUdpTunnel")
					if err := Tun.UdpTunnel.SetUdpTunnel(*men.Response.Content.UdpChannelDetails); err != nil {
						timeout++
						LogDebug.Print(err)
					}
				} else if cont.Pong != nil {
					Tun.LastPong = uint64(time.Now().UnixMilli())
					if cont.Pong.ClientAddr.Compare(Tun.ControlChannel.Conn.Pong.ClientAddr.AddrPort) != 0 {
						LogDebug.Printf("Client IP changed: %q -> %q\n", cont.Pong.ClientAddr, Tun.ControlChannel.Conn.Pong.ClientAddr.AddrPort)
					}
				} else if cont.Unauthorized {
					LogDebug.Panicln("unauthorized, check token or reload agent")
					Tun.ControlChannel.ForceEpired = true
					return nil, fmt.Errorf("unauthorized, check token or reload agent")
				} else {
					LogDebug.Printf("got response")
					d , _ := json.MarshalIndent(men, "", "  ")
					LogDebug.Printf(string(d))
				}
			}
		}
	}

	if Tun.LastPong != 0 && uint64(time.Now().UnixMilli())-Tun.LastPong > 6_000 {
		LogDebug.Println("timeout waiting for pong")
		Tun.LastPong = 0
		Tun.ControlChannel.ForceEpired = true
	}

	return nil, nil
}
