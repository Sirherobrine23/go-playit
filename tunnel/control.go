package tunnel

import (
	"bytes"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type AuthenticatedControl struct {
	ApiClient   api.Api
	Conn        ConnectedControl
	CurrentPing *uint32
	LastPong    Pong
	ForceEpired bool
	Registered    AgentRegistered
	Buff        []byte
}

func (Auth *AuthenticatedControl) Send(Req ControlRpcMessage[MessageEncoding]) error {
	Auth.Buff = []byte{}
	if err := Req.WriteTo(bytes.NewBuffer(Auth.Buff)); err != nil {
		return err
	} else if _, err := Auth.Conn.Udp.Write(Auth.Buff); err != nil {
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
	return Auth.LastPong.ClientAddr.Compare(Auth.LastPong.ClientAddr.AddrPort) == 0
}

func (Auth *AuthenticatedControl) IsIspired() bool {
	return Auth.ForceEpired || Auth.LastPong.SessionExpireAt == nil || Auth.FlowChanged()
}

func (Auth *AuthenticatedControl) IntoRequiresAuth() *ConnectedControl {
	return &ConnectedControl{
		ControlAddr: Auth.Conn.ControlAddr,
		Udp: Auth.Conn.Udp,
		Pong: Auth.LastPong,
	}
}

func (Auth *AuthenticatedControl) RecFeedMsg() (*ControlFeed, error) {
	Auth.Buff = make([]byte, 1024)
	if _, err := Auth.Conn.Udp.Read(Auth.Buff); err != nil {
		return nil, err
	}
	var feed ControlFeed
	if err := feed.ReadFrom(bytes.NewBuffer(Auth.Buff)); err != nil {
		return nil, err
	}

	if feed.Response != nil {
		if feed.Response.Content != nil {
			if feed.Response.Content.AgentRegistered != nil {
				Auth.Registered = *feed.Response.Content.AgentRegistered
			} else if feed.Response.Content.Pong != nil {
				CurrentPing := uint32(feed.Response.Content.Pong.RequestNow - uint64(time.Now().UnixMilli()))
				Auth.CurrentPing = &CurrentPing
				Auth.LastPong = *feed.Response.Content.Pong
				if feed.Response.Content.Pong.SessionExpireAt != nil {
					Auth.Registered.ExpiresAt = *feed.Response.Content.Pong.SessionExpireAt
				}
			}
		}

	}
	return &feed, nil
}

func (Auth *AuthenticatedControl) Authenticate() (*AuthenticatedControl, error) {
	conn := ConnectedControl{
		ControlAddr: Auth.Conn.ControlAddr,
		Udp: Auth.Conn.Udp,
		Pong: Auth.LastPong,
	}
	return conn.Authenticate(Auth.ApiClient)
}