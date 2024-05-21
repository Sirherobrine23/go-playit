package tunnel

import (
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
	UdpTunnel          *UdpTunnel
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
	var err error
	Tun.UdpTunnel, err = NewUdpTunnel()
	if err != nil {
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
	routs, err := Tun.ApiClaim.AgentRoutings(nil)
	if err != nil {
		return false, err
	}
	addresses := []netip.AddrPort{}
	for _, v := range append(routs.Targets6, routs.Targets4...) {
		addresses = append(addresses, netip.AddrPortFrom(v, 5525))
	}
	skip := true
	for _, addr := range addresses {
		if !slices.Contains(Tun.lastControlTargets, addr) {
			skip = false
		}
	}
	if skip {
		return false, nil
	}
	tun2 := SimplesTunnel{
		ApiClaim: Tun.ApiClaim,
	}
	if err := tun2.Setup(); err != nil {
		return false, err
	}
	updated, err := Tun.UpdateControlAddr(tun2.ControlChannel.Conn)
	if err != nil {
		return updated, err
	}
	Tun.lastControlTargets = addresses
	return false, nil
}

func (Tun *SimplesTunnel) UpdateControlAddr(conncted ConnectedControl) (ok bool, err error) {
	if conncted.ControlAddr.Compare(Tun.ControlAddr) == 0 {
		return
	}
	var controlChannel *AuthenticatedControl
	controlChannel, err = conncted.Authenticate(Tun.ApiClaim)
	if err != nil {
		return
	}
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
		fmt.Println("Creating new controller channel...")
		newControlChannel, err := Tun.ControlChannel.Authenticate()
		if err != nil {
			time.Sleep(time.Second * 2)
			return nil, err
		}
		Tun.ControlChannel = newControlChannel
	}

	now := uint64(time.Now().UnixMilli())
	if now-Tun.LastPing > 1_000 {
		Tun.LastPing = uint64(now)
		if err := Tun.ControlChannel.SendPing(200, time.UnixMilli(int64(now))); err != nil {
			return nil, err
		}
	}

	if Tun.UdpTunnel.RequiresAuth() {
		if 5_000 < now-Tun.LastUdpAuth {
			Tun.LastUdpAuth = uint64(now)
			if err := Tun.ControlChannel.SendSetupUDPChannel(9000); err != nil {
				fmt.Println(err)
			}
		}
	} else if Tun.UdpTunnel.RequireResend() {
		if 1_000 < now-Tun.LastUdpAuth {
			Tun.LastUdpAuth = uint64(now)
			if _, err := Tun.UdpTunnel.ResendToken(); err != nil {
				return nil, err
			}
		}

		timeTillExpire := func(x, y uint64) uint64 {
			if x > y {
				return y
			}
			return x
		}(Tun.ControlChannel.Registered.ExpiresAt, uint64(now))
		if 10_000 < now-Tun.LastKeepAlive && timeTillExpire < 30_000 {
			Tun.LastKeepAlive = now
			if err := Tun.ControlChannel.SendKeepAlive(100); err != nil {
				return nil, err
			} else if err := Tun.ControlChannel.SendSetupUDPChannel(1); err != nil {
				return nil, err
			}
		}

		timeout := 0
		for range 30 {
			if timeout >= 10 {
				return nil, nil
			}
			fmt.Println("RX Feed message")
			men, err := Tun.ControlChannel.RecFeedMsg()
			if err != nil {
				fmt.Println(err)
				timeout++
			}
			if men.NewClient != nil {
				return men.NewClient, nil
			} else if men.Response != nil {
				cont := men.Response.Content
				if cont.UdpChannelDetails != nil {
					fmt.Println("SetUdpTunnel")
					if err := Tun.UdpTunnel.SetUdpTunnel(*men.Response.Content.UdpChannelDetails); err != nil {
						return nil, err
					}
				} else if cont.Pong != nil {
					Tun.LastPong = uint64(time.Now().UnixMilli())
					// if cont.Pong.ClientAddr.Compare(Tun.ControlChannel.Conn.Pong.ClientAddr.AddrPort) != 0 {
					// }
				} else if cont.Unauthorized {
					Tun.ControlChannel.ForceEpired = true
					return nil, fmt.Errorf("unauthorized, check token or reload agent")
				}
			}
		}
	}

	if Tun.LastPong != 0 && uint64(time.Now().UnixMilli())-Tun.LastPong > 6_000 {
		fmt.Println("timeout waiting for pong")
		Tun.LastPong = 0
		Tun.ControlChannel.ForceEpired = true
	}

	return nil, nil
}
