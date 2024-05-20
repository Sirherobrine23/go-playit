package tunnel

import (
	"bytes"
	"fmt"
	"net"
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

func (Tun *SimplesTunnel) Setup() error {
	controls, err := Tun.ApiClaim.AgentRoutings(nil)
	if err != nil {
		return err
	}

	Tun.lastControlTargets = []netip.AddrPort{}
	for _, v := range append(controls.Targets6, controls.Targets4...) {
		Tun.lastControlTargets = append(Tun.lastControlTargets, netip.AddrPortFrom(v, 5525))
	}

	for _, Addr := range Tun.lastControlTargets {
		Tun.ControlAddr = Addr
		conn, err := net.Dial("udp", Tun.ControlAddr.String())
		if err != nil {
			continue
		}

		// Make initial ping
		buffer := bytes.NewBuffer([]byte{})
		err = (&ControlRpcMessage[*ControlRequest]{
			RequestID: 1,
			Content: &ControlRequest{
				Ping: &Ping{
					Now:         time.Now(),
					CurrentPing: nil,
					SessionID:   nil,
				},
			},
		}).WriteTo(buffer)
		if err != nil {
			conn.Close()
			return err
		}

		// Write initial ping
		_, err = buffer.WriteTo(conn)
		if err != nil {
			conn.Close()
			return err
		}

		// Get response from ping
		buffer.Reset()
		conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		bytesRead := make([]byte, 2048)
		_, err = conn.Read(bytesRead)
		if err != nil {
			conn.Close()
			return err
		}

		// Parse reponse pong
		reader := bytes.NewReader(bytesRead)
		res := ControlFeed{}
		if err := res.ReadFrom(reader); err != nil {
			conn.Close()
			return err
		} else if res.Response == nil || res.Response.RequestID != 1 {
			conn.Close()
			return fmt.Errorf("got response with unexpected request_id")
		} else if res.Response == nil {
			conn.Close()
			return fmt.Errorf("expected controlRequest")
		}

		Response := res.Response.Content
		if Response.Pong == nil {
			conn.Close()
			return fmt.Errorf("expected pong got other response")
		}

		connControl := ConnectedControl{
			ControlAddr: Tun.ControlAddr,
			Udp:         conn,
			Pong:        *Response.Pong,
		}

		Tun.ControlChannel, err = connControl.Authenticate(Tun.ApiClaim)
		return err
	}

	return fmt.Errorf("cannot make UDP tunnel to playit controller, check you internet conenction")
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
	ok = false
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

func (Tun *SimplesTunnel) Update() *NewClient {
	if Tun.ControlChannel.IsIspired() {
		newControlChannel, err := Tun.ControlChannel.Authenticate()
		if err != nil {
			time.Sleep(time.Second * 2)
			return nil
		}
		Tun.ControlChannel = newControlChannel
	}

	now := uint64(time.Now().UnixMilli())
	if now-Tun.LastPing > 1_000 {
		Tun.LastPing = uint64(now)
		if err := Tun.ControlChannel.SendPing(200, time.UnixMilli(int64(now))); err != nil {
			fmt.Println(err)
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
				fmt.Println(err)
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
				fmt.Println(err)
			} else if err := Tun.ControlChannel.SendSetupUDPChannel(1); err != nil {
				fmt.Println(err)
			}
		}

		timeout := 0
		for range 30 {
			if timeout >= 10 {
				return nil
			}
			men, err := Tun.ControlChannel.RecFeedMsg()
			if err != nil {
				fmt.Println(err)
				timeout++
			}
			if men.NewClient != nil {
				return men.NewClient
			} else if men.Response != nil {
				cont := men.Response.Content
				if cont.UdpChannelDetails != nil {
					if err := Tun.UdpTunnel.SetUdpTunnel(*men.Response.Content.UdpChannelDetails); err != nil {
						panic(err)
					}
				} else if cont.Pong != nil {
					Tun.LastPong = uint64(time.Now().UnixMilli())
					// if cont.Pong.ClientAddr.Compare(Tun.ControlChannel.Conn.Pong.ClientAddr.AddrPort) != 0 {
					// }
				} else if cont.Unauthorized {
					Tun.ControlChannel.ForceEpired = true
				}
			}
		}
	}

	if Tun.LastPong != 0 && uint64(time.Now().UnixMilli())-Tun.LastPong > 6_000 {
		// fmt.Println("timeout waiting for pong")
		Tun.LastPong = 0
		Tun.ControlChannel.ForceEpired = true
	}

	return nil
}
