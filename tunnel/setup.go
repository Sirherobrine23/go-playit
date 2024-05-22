package tunnel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type SetupFindSuitableChannel struct {
	Address []netip.AddrPort
}

func (Setup *SetupFindSuitableChannel) Setup() (*ConnectedControl, error) {
	for _, Addr := range Setup.Address {
		network := "udp6"
		if Addr.Addr().Is4() && !Addr.Addr().Is4In6() {
			network = "udp4"
		}
		conn, err := net.ListenUDP(network, nil)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		for range 3 {
			// Make initial ping
			buffer := bytes.NewBuffer([]byte{})
			if err = (&ControlRpcMessage[*ControlRequest]{
				RequestID: 1,
				Content: &ControlRequest{
					Ping: &Ping{
						Now:         time.Now(),
						CurrentPing: nil,
						SessionID:   nil,
					},
				},
			}).WriteTo(buffer); err != nil {
				conn.Close()
				return nil, err
			}

			// Write initial ping
			_, err = conn.WriteToUDP(buffer.Bytes(), net.UDPAddrFromAddrPort(Addr))
			if err != nil {
				conn.Close()
				break
			}

			for range 5 {
				buff := make([]byte, 2048)
				if err = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 5)); err != nil {
					return nil, err
				}
				bytesSize, peer, err := conn.ReadFrom(buff)
				if err != nil {
					if netErr, isNet := err.(net.Error); isNet {
						if netErr.Timeout() {
							continue
						}
					}
					return nil, err
				} else if peer.String() != Addr.String() {
					continue
				}

				buff = buff[:bytesSize]
				var feed ControlFeed
				if err := feed.ReadFrom(bytes.NewReader(buff)); err != nil {
					return nil, err
				} else if feed.Response == nil {
					return nil, fmt.Errorf("unexpected control feed")
				}

				msg := feed.Response
				if msg.RequestID != 1 {
					continue
				} else if msg.Content.Pong == nil {
					return nil, fmt.Errorf("expected pong got other response")
				}
				return &ConnectedControl{
					ControlAddr: Addr,
					Udp:         conn,
					Pong:        msg.Content.Pong,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot make UDP tunnel to playit controller, check you internet conenction")
}

type ConnectedControl struct {
	ControlAddr netip.AddrPort
	Udp         *net.UDPConn
	Pong        *Pong
}

func (Control *ConnectedControl) Authenticate(Api api.Api) (*AuthenticatedControl, error) {
	if !Control.Pong.ClientAddr.AddrPort.IsValid() {
		return nil, fmt.Errorf("invalid pong Client address")
	} else if !Control.Pong.TunnelAddr.AddrPort.IsValid() {
		return nil, fmt.Errorf("invalid pong Tunnel address")
	}

	LogDebug.Println("Registring agent proto")
	tk, err := Api.ProtoRegisterRegister(Control.Pong.ClientAddr.AddrPort, Control.Pong.TunnelAddr.AddrPort)
	if err != nil {
		LogDebug.Println("failed to sign and register")
		return nil, err
	}

	tkBytes, err := hex.DecodeString(tk)
	if err != nil {
		return nil, err
	}

	for range 5 {
		buffer := bytes.NewBuffer([]byte{})
		if err := (&ControlRpcMessage[*RawSlice]{
			RequestID: 10,
			Content: &RawSlice{
				Buff: tkBytes,
			},
		}).WriteTo(buffer); err != nil {
			return nil, err
		}
		_, err := Control.Udp.WriteTo(buffer.Bytes(), net.UDPAddrFromAddrPort(Control.ControlAddr))
		if err != nil {
			return nil, err
		}

		for range 5 {
			reciver := append(buffer.Bytes(), make([]byte, 1024)...)
			Control.Udp.SetReadDeadline(time.Now().Add(time.Millisecond * 5))
			recSize, remote, err := Control.Udp.ReadFrom(reciver)
			if err != nil {
				if errNet, isNet := err.(net.Error); isNet {
					if errNet.Timeout() {
						LogDebug.Println("Timeout")
						break
					}
				}
				return nil, err
			} else if remote.String() != Control.ControlAddr.String() {
				LogDebug.Println("got response not from tunnel server")
				continue
			}

			feed := &ControlFeed{}
			if err = feed.ReadFrom(bytes.NewReader(reciver[:recSize])); err != nil {
				LogDebug.Println("failed to read response from tunnel")
				return nil, err
			} else if feed.Response.RequestID != 10 {
				LogDebug.Println("got response for different request")
				continue
			} else if feed.Response == nil || feed.Response.Content == nil {
				LogDebug.Println("feed response or Response content is empty")
				return nil, fmt.Errorf("cannot get response")
			}

			controlRes := feed.Response.Content
			if controlRes.RequestQueued {
				LogDebug.Println("register queued, waiting 1s")
				time.Sleep(time.Second)
				continue
			} else if controlRes.InvalidSignature {
				return nil, fmt.Errorf("register return invalid signature")
			} else if controlRes.Unauthorized {
				return nil, fmt.Errorf("unauthorized")
			} else if controlRes.AgentRegistered != nil {
				return &AuthenticatedControl{
					ApiClient:   Api,
					Conn:        *Control,
					LastPong:    *Control.Pong,
					CurrentPing: nil,
					Registered:  *controlRes.AgentRegistered,
					Buff:        reciver[recSize:],
					ForceEpired: false,
				}, nil
			}
			LogDebug.Println("expected AgentRegistered but got something else")
		}
	}
	return nil, fmt.Errorf("failed1 to connect agent")
}
