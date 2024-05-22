package tunnel

import (
	"bytes"
	"fmt"
	"net/netip"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type AuthenticatedControl struct {
	ApiClient   api.Api
	Conn        ConnectedControl
	CurrentPing *uint32
	LastPong    Pong
	ForceEpired bool
	Registered  AgentRegistered
	Buff        []byte
}

func (Auth *AuthenticatedControl) Send(Req ControlRpcMessage[MessageEncoding]) error {
	Auth.Buff = []byte{}
	bufio := bytes.NewBuffer(Auth.Buff)
	if err := Req.WriteTo(bufio); err != nil {
		return err
	}
	Auth.Buff = bufio.Bytes()
	_, err := Auth.Conn.Udp.WriteToUDPAddrPort(Auth.Buff, Auth.Conn.ControlAddr)
	if err != nil {
		return err
	}
	return nil
}

func (Auth *AuthenticatedControl) SendKeepAlive(RequestID uint64) error {
	return Auth.Send(ControlRpcMessage[MessageEncoding]{
		RequestID: RequestID,
		Content: &ControlRequest{
			AgentKeepAlive: &Auth.Registered.ID,
		},
	})
}

func (Auth *AuthenticatedControl) SendSetupUDPChannel(RequestID uint64) error {
	return Auth.Send(ControlRpcMessage[MessageEncoding]{
		RequestID: RequestID,
		Content: &ControlRequest{
			SetupUdpChannel: &Auth.Registered.ID,
		},
	})
}

func (Auth *AuthenticatedControl) SendPing(RequestID uint64, Now time.Time) error {
	return Auth.Send(ControlRpcMessage[MessageEncoding]{
		RequestID: RequestID,
		Content: &ControlRequest{
			Ping: &Ping{
				Now:         Now,
				CurrentPing: Auth.CurrentPing,
				SessionID:   &Auth.Registered.ID,
			},
		},
	})
}

func (Auth *AuthenticatedControl) FlowChanged() bool {
	return Auth.LastPong.ClientAddr.Compare(Auth.LastPong.ClientAddr.AddrPort) != 0
}

func (Auth *AuthenticatedControl) IsIspired() bool {
	return Auth.ForceEpired || Auth.LastPong.SessionExpireAt == nil || Auth.FlowChanged()
}

func (Auth *AuthenticatedControl) IntoRequiresAuth() *ConnectedControl {
	return &ConnectedControl{
		ControlAddr: Auth.Conn.ControlAddr,
		Udp:         Auth.Conn.Udp,
		Pong:        &Auth.LastPong,
	}
}

type InvalidRemote struct {
	Expected, Got netip.AddrPort
}

func (a InvalidRemote) Error() string {
	return fmt.Sprintf("expected %s, got %s", a.Expected.String(), a.Got.String())
}

func (Auth *AuthenticatedControl) RecFeedMsg() (*ControlFeed, error) {
	Auth.Buff = append(Auth.Buff, make([]byte, 1024)...)
	size, remote, err := Auth.Conn.Udp.ReadFromUDP(Auth.Buff)
	LogDebug.Println(size, remote, err)
	if err != nil {
		return nil, err
	} else if remote.AddrPort().Compare(Auth.Conn.ControlAddr) != 0 {
		return nil, InvalidRemote{Expected: Auth.Conn.ControlAddr, Got: remote.AddrPort()}
	}

	var feed ControlFeed
	if err := feed.ReadFrom(bytes.NewBuffer(Auth.Buff[size:])); err != nil {
		return nil, err
	}

	if feed.Response != nil {
		if feed.Response.Content != nil {
			if feed.Response.Content.AgentRegistered != nil {
				LogDebug.Println("agent registred")
				LogDebug.Printf("%+v\n", feed.Response.Content.AgentRegistered)
				Auth.Registered = *feed.Response.Content.AgentRegistered
			} else if feed.Response.Content.Pong != nil {
				CurrentPing := uint32(feed.Response.Content.Pong.RequestNow - uint64(time.Now().UnixMilli()))
				Auth.CurrentPing = &CurrentPing
				Auth.LastPong = *feed.Response.Content.Pong
				if feed.Response.Content.Pong.SessionExpireAt != nil {
					Auth.Registered.ExpiresAt = time.UnixMilli(int64(*feed.Response.Content.Pong.SessionExpireAt))
				}
			}
		}

	}
	return &feed, nil
}

func (Auth *AuthenticatedControl) Authenticate() error {
	conn, err := (&ConnectedControl{
		ControlAddr: Auth.Conn.ControlAddr,
		Udp:         Auth.Conn.Udp,
		Pong:        &Auth.LastPong,
	}).Authenticate(Auth.ApiClient)
	if err != nil {
		return err
	}
	Auth.Buff = conn.Buff
	Auth.Conn = conn.Conn
	Auth.CurrentPing = conn.CurrentPing
	Auth.LastPong = conn.LastPong
	Auth.Registered = conn.Registered
	return nil
}
