package tunnel

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

type AuthenticatedControl struct {
	Api         api.Api
	Conn        ConnectedControl
	LastPong    proto.Pong
	Registered  proto.AgentRegistered
	buffer      *bytes.Buffer
	ForceExpire bool
	CurrentPing *uint64
}

func (self *AuthenticatedControl) SendKeepAlive(requestId uint64) error {
	return self.Send(proto.ControlRpcMessage[*proto.ControlRequest]{
		requestId,
		&proto.ControlRequest{
			AgentKeepAlive: &self.Registered.Id,
		},
	})
}

func (self *AuthenticatedControl) SendSetupUdpChannel(requestId uint64) error {
	return self.Send(proto.ControlRpcMessage[*proto.ControlRequest]{
		requestId,
		&proto.ControlRequest{
			SetupUdpChannel: &self.Registered.Id,
		},
	})
}

func (self *AuthenticatedControl) SendPing(requestId uint64, Now time.Time) error {
	return self.Send(proto.ControlRpcMessage[*proto.ControlRequest]{
		requestId,
		&proto.ControlRequest{
			Ping: &proto.Ping{Now, self.CurrentPing, &self.Registered.Id},
		},
	})
}

func (self *AuthenticatedControl) GetExpireAt() time.Time {
	return self.Registered.ExpiresAt
}

func (self *AuthenticatedControl) IsExpired() bool {
	return self.ForceExpire || self.LastPong.SessionExpireAt == nil || self.FlowChanged()
}

func (self *AuthenticatedControl) SetExpired() {
	self.ForceExpire = true
}

func (self *AuthenticatedControl) FlowChanged() bool {
	return self.Conn.Pong.ClientAddr.Compare(self.LastPong.ClientAddr) != 0
}

func (self *AuthenticatedControl) Send(req proto.ControlRpcMessage[*proto.ControlRequest]) error {
	self.buffer.Reset()
	if err := req.WriteTo(self.buffer); err != nil {
		return err
	} else if _, err := self.Conn.Udp.WriteTo(self.buffer.Bytes(), net.UDPAddrFromAddrPort(self.Conn.ControlAddr)); err != nil {
		return err
	}
	return nil
}

func (self *AuthenticatedControl) IntoRequireAuth() ConnectedControl {
	return ConnectedControl{
		ControlAddr: self.Conn.ControlAddr,
		Udp: self.Conn.Udp,
		Pong: self.LastPong,
	}
}

func (self *AuthenticatedControl) Authenticate() error {
	conn := self.IntoRequireAuth()
	var err error
	if *self, err = conn.Authenticate(self.Api); err != nil {
		return err
	}
	return nil
}

func (self *AuthenticatedControl) RecvFeedMsg() (proto.ControlFeed, error) {
	buff := make([]byte, 1024)
	size, remote, err := self.Conn.Udp.ReadFromUDPAddrPort(buff)
	self.buffer.Reset()
	self.buffer.Write(buff[:size])
	if err != nil {
		return proto.ControlFeed{}, err
	} else if remote.Compare(self.Conn.ControlAddr) != 0 {
		return proto.ControlFeed{}, fmt.Errorf("invalid remote, expected %q got %q", remote.String(), self.Conn.ControlAddr.String())
	}
	feed := proto.ControlFeed{}
	if err := feed.ReadFrom(self.buffer); err != nil {
		return proto.ControlFeed{}, err
	}
	if feed.Response != nil {
		res := feed.Response
		if registered := res.Content.AgentRegistered; registered != nil {
			self.Registered = *registered
		}
		if pong := res.Content.Pong; pong != nil {
			*self.CurrentPing = uint64(time.Now().UnixMilli() - pong.RequestNow.UnixMilli())
			self.LastPong = *pong
		}
	}
	return feed, nil
}