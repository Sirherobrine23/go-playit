package tunnel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

type SetupFindSuitableChannel struct {
	options []netip.AddrPort
}

func (self *SetupFindSuitableChannel) Setup() (ConnectedControl, error) {
	for _, addr := range self.options {
		var (
			err    error
			socket *net.UDPConn
		)
		isIPv6 := addr.Addr().Is6()
		if isIPv6 {
			if socket, err = net.ListenUDP("udp6", nil); err != nil {
				continue // Next address to listen
			}
		} else {
			if socket, err = net.ListenUDP("udp4", nil); err != nil {
				continue // Next address to listen
			}
		}
		var attempts int
		if attempts = 3; isIPv6 {
			attempts = 1
		}
		for range attempts {
			buffer := new(bytes.Buffer)
			if err := (&proto.ControlRpcMessage[*proto.ControlRequest]{
				RequestID: 1,
				Content: &proto.ControlRequest{
					Ping: &proto.Ping{
						Now:         time.Now(),
						CurrentPing: nil,
						SessionId:   nil,
					},
				},
			}).WriteTo(buffer); err != nil {
				continue
			}
			if _, err := socket.WriteTo(buffer.Bytes(), net.UDPAddrFromAddrPort(addr)); err != nil {
				break
			}

			buffer.Reset()
			var waits int
			if waits = 5; isIPv6 {
				waits = 3
			}
			for range waits {
				buff := make([]byte, 1024)
				socket.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				size, peer, err := socket.ReadFrom(buff)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						continue
					}
					break
				} else if peer.String() != addr.String() {
					continue
				}
				buffer = bytes.NewBuffer(buff[:size])
				feed := proto.ControlFeed{}
				if err := feed.ReadFrom(buffer); err != nil {
					break
				} else if feed.Response == nil {
					break
				} else if feed.Response.RequestID != 1 {
					break
				} else if feed.Response.Content.Pong == nil {
					break
				}
				return ConnectedControl{addr, *socket, *feed.Response.Content.Pong}, nil
			}
		}
		socket.Close()
	}
	return ConnectedControl{}, fmt.Errorf("failed to connectans setup initial connection")
}

type ConnectedControl struct {
	ControlAddr netip.AddrPort
	Udp         net.UDPConn
	Pong        proto.Pong
}

func (self *ConnectedControl) Authenticate(Api api.Api) (AuthenticatedControl, error) {
	key, err := Api.ProtoRegisterRegister(self.Pong.ClientAddr, self.Pong.TunnelAddr)
	if err != nil {
		return AuthenticatedControl{}, err
	}
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return AuthenticatedControl{}, err
	}
	for range 5 {
		buffer := new(bytes.Buffer)
		if err := (&proto.ControlRpcMessage[proto.RawSlice]{
			RequestID: 10,
			Content:   proto.RawSlice(keyBytes),
		}).WriteTo(buffer); err != nil {
			return AuthenticatedControl{}, err
		} else if _, err := self.Udp.WriteTo(buffer.Bytes(), net.UDPAddrFromAddrPort(self.ControlAddr)); err != nil {
			return AuthenticatedControl{}, err
		}
		for range 5 {
			buff := make([]byte, 1024)
			self.Udp.SetReadDeadline(time.Now().Add(time.Millisecond * 5))
			size, remote, err := self.Udp.ReadFromUDPAddrPort(buff)
			if err != nil {
				if at, ok := err.(net.Error); ok && at.Timeout() {
					continue
				}
				break
			} else if self.ControlAddr.Compare(remote) != 0 {
				continue
			}
			buffer.Reset()
			buffer.Write(buff[:size]) // Write only reader data
			var feed proto.ControlFeed
			if err := feed.ReadFrom(buffer); err != nil {
				continue
			}

			if response := feed.Response; response != nil {
				if response.RequestID != 10 {
					continue
				}
				if content := response.Content; content.RequestQueued {
					time.Sleep(time.Second) // Sleep to wait register
					break
				} else if content.InvalidSignature {
					return AuthenticatedControl{}, fmt.Errorf("invalid signature")
				} else if content.Unauthorized {
					return AuthenticatedControl{}, fmt.Errorf("unauthorized")
				} else if registered := content.AgentRegistered; registered != nil {
					return AuthenticatedControl{
						Api:         Api,
						Conn:        *self,
						LastPong:    self.Pong,
						Registered:  *registered,
						buffer:      buffer,
						CurrentPing: nil,
						ForceExpire: false,
					}, nil
				}
			}
		}
	}
	return AuthenticatedControl{}, fmt.Errorf("failed to connect and authenticate")
}
