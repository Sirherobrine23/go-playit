package tunnel

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

func getControlAddresses(api api.Api) ([]netip.AddrPort, error) {
	routing, err := api.AgentRoutings(nil)
	if err != nil {
		return nil, err
	}
	addresses := []netip.AddrPort{}
	for _, ipd := range append(routing.Targets6, routing.Targets4...) {
		addresses = append(addresses, netip.AddrPortFrom(ipd, 5525))
	}
	return addresses, nil
}

type SimpleTunnel struct {
	api                                            api.Api
	controlAddr                                    netip.AddrPort
	controlChannel                                 AuthenticatedControl
	udpTunnel                                      UdpTunnel
	lastKeepAlive, lastPing, lastPong, lastUdpAuth time.Time
	lastControlTargets                             []netip.AddrPort
}

func NewSimpleTunnel(Api api.Api) SimpleTunnel {
	return SimpleTunnel{
		api: Api,
	}
}

func (self *SimpleTunnel) Setup() error {
	udpTunnel := UdpTunnel{}
	if err := AssignUdpTunnel(&udpTunnel); err != nil {
		return err
	}

	addresses, err := getControlAddresses(self.api)
	if err != nil {
		return err
	}
	setup, err := (&SetupFindSuitableChannel{addresses}).Setup()
	if err != nil {
		return err
	}

	controlChannel, err := setup.Authenticate(self.api)
	if err != nil {
		return err
	}

	self.lastControlTargets = addresses
	self.controlAddr = setup.ControlAddr
	self.controlChannel = controlChannel
	self.udpTunnel = udpTunnel
	self.lastKeepAlive = time.UnixMilli(0)
	self.lastPing = time.UnixMilli(0)
	self.lastPong = time.UnixMilli(0)
	self.lastUdpAuth = time.UnixMilli(0)
	return nil
}

func (self *SimpleTunnel) ReloadControlAddr() (bool, error) {
	addresses, err := getControlAddresses(self.api)
	if err != nil {
		return false, err
	} else if slices.ContainsFunc(self.lastControlTargets, func(e1 netip.AddrPort) bool {
		return !slices.Contains(addresses, e1)
	}) {
		return false, nil
	}
	setup, err := (&SetupFindSuitableChannel{addresses}).Setup()
	if err != nil {
		return false, err
	}
	updated, err := self.UpdateControlAddr(setup)
	if err == nil {
		self.lastControlTargets = addresses
	}
	return updated, err
}
func (self *SimpleTunnel) UpdateControlAddr(connected ConnectedControl) (bool, error) {
	newControlAddr := connected.ControlAddr
	if self.controlAddr.Compare(newControlAddr) == 0 {
		return false, nil
	}
	controlChannel, err := connected.Authenticate(self.api)
	if err != nil {
		return false, err
	}
	self.controlChannel = controlChannel
	self.controlAddr = newControlAddr
	self.lastPing, self.lastKeepAlive, self.lastUdpAuth = time.UnixMilli(0), time.UnixMilli(0), time.UnixMilli(0)
	self.udpTunnel.InvalidateSession()
	return true, nil
}

func (self *SimpleTunnel) UdpTunnel() UdpTunnel {
	return self.udpTunnel
}

func (self *SimpleTunnel) Update() *proto.NewClient {
	if self.controlChannel.IsExpired() {
		auth, err := self.controlChannel.Authenticate()
		if err != nil {
			time.Sleep(time.Second * 2)
			return nil
		}
		self.controlChannel = auth
	}

	now := time.Now()
	if now.UnixMilli() - self.lastPing.UnixMilli() > 1_000 {
		self.lastPing = now
		if err := self.controlChannel.SendPing(200, now); err != nil {}
	}
	if self.udpTunnel.RequiresAuth() {
		if 5_000 < now.UnixMilli() - self.lastUdpAuth.UnixMilli() {
			self.lastUdpAuth = now
			if err := self.controlChannel.SendSetupUdpChannel(9_000); err != nil {}
		}
	} else if self.udpTunnel.RequireResend() {
		if 1_000 < now.UnixMilli() - self.lastUdpAuth.UnixMilli() {
			self.lastUdpAuth = now
			if _, err := self.udpTunnel.ResendToken(); err != nil {}
		}
	}

	timeTillExpire := max(self.controlChannel.GetExpireAt().UnixMilli(), now.UnixMilli()) - now.UnixMilli()
	if 10_000 < now.UnixMilli() - self.lastKeepAlive.UnixMilli() && timeTillExpire < 30_000 {
		self.lastKeepAlive = now
		if err := self.controlChannel.SendKeepAlive(100); err != nil {}
		if err := self.controlChannel.SendSetupUdpChannel(1); err != nil {}
	}

	for range 30 {
		feed, err := self.controlChannel.RecvFeedMsg()
		if err != nil {
			fmt.Println(err)
			continue
		}
		d,_:=json.MarshalIndent(feed, "", "  ")
		fmt.Printf("SimTunne: %s\n", string(d))
		if newClient := feed.NewClient; newClient != nil {
			return newClient
		} else if msg := feed.Response; msg != nil {
			if content := msg.Content; content != nil {
				if details := content.UdpChannelDetails; details != nil {
					if err := self.udpTunnel.SetUdpTunnel(*details); err != nil {
						panic(err)
					}
				} else if content.Unauthorized {
					self.controlChannel.SetExpired()
				} else if pong := content.Pong; pong != nil {
					self.lastPong = time.Now()
					if pong.ClientAddr.Compare(self.controlChannel.Conn.Pong.ClientAddr) != 0 {
						fmt.Println("client ip changed", pong.ClientAddr.String(), self.controlChannel.Conn.Pong.ClientAddr.String())
					}
				}
			}
		}
	}
	if self.lastPong.UnixMilli() != 0 && time.Now().UnixMilli() - self.lastPong.UnixMilli() > 6_000 {
		self.lastPong = time.UnixMilli(0)
		self.controlChannel.SetExpired()
	}
	return nil
}