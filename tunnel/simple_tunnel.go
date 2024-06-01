package tunnel

import (
	"net"
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
	ControlChannel                                 AuthenticatedControl
	udpTunnel                                      *UdpTunnel
	lastKeepAlive, lastPing, lastPong, lastUdpAuth time.Time
	lastControlTargets                             []netip.AddrPort
}

func NewSimpleTunnel(Api api.Api) SimpleTunnel {
	return SimpleTunnel{
		api: Api,
	}
}

func (self *SimpleTunnel) Setup() error {
	udpTunnel := new(UdpTunnel)
	if err := AssignUdpTunnel(udpTunnel); err != nil {
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
	self.ControlChannel = controlChannel
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
	self.ControlChannel = controlChannel
	self.controlAddr = newControlAddr
	self.lastPing, self.lastKeepAlive, self.lastUdpAuth = time.UnixMilli(0), time.UnixMilli(0), time.UnixMilli(0)
	self.udpTunnel.InvalidateSession()
	return true, nil
}

func (self *SimpleTunnel) UdpTunnel() *UdpTunnel {
	return self.udpTunnel
}

func (self *SimpleTunnel) Update() (*proto.NewClient, error) {
	if self.ControlChannel.IsExpired() {
		auth, err := self.ControlChannel.Authenticate()
		if err != nil {
			time.Sleep(time.Second * 2)
			return nil, err
		}
		self.ControlChannel = auth
	}

	now := time.Now()
	if now.UnixMilli()-self.lastPing.UnixMilli() > 1_000 {
		self.lastPing = now
		if err := self.ControlChannel.Ping(200, now); err != nil {
			debug.Printf("Update: %s\n", err.Error())
			return nil, err
		}
	}

	if self.udpTunnel.RequiresAuth() {
		if 5_000 < now.UnixMilli()-self.lastUdpAuth.UnixMilli() {
			self.lastUdpAuth = now
			if err := self.ControlChannel.SetupUdpChannel(9_000); err != nil {
				debug.Printf("Update: %s\n", err.Error())
				return nil, err
			}
		}
	} else if self.udpTunnel.RequireResend() {
		if 1_000 < now.UnixMilli()-self.lastUdpAuth.UnixMilli() {
			self.lastUdpAuth = now
			if _, err := self.udpTunnel.ResendToken(); err != nil {
				return nil, err
			}
		}
	}

	timeTillExpire := max(self.ControlChannel.GetExpireAt().UnixMilli(), now.UnixMilli()) - now.UnixMilli()
	if 10_000 < now.UnixMilli()-self.lastKeepAlive.UnixMilli() && timeTillExpire < 30_000 {
		self.lastKeepAlive = now
		if err := self.ControlChannel.SendKeepAlive(100); err != nil {
			return nil, err
		} else if err := self.ControlChannel.SendSetupUdpChannel(1); err != nil {
			return nil, err
		}
	}

	for range 80 {
		feed, err := self.ControlChannel.RecvFeedMsg()
		if err != nil {
			if es, is := err.(net.Error); is && !es.Timeout() {
				debug.Printf("RecvFeedMsg error: %s\n", err.Error())
				return nil, err
			}
			continue
		}
		if newClient := feed.NewClient; newClient != nil {
			return newClient, nil
		} else if msg := feed.Response; msg != nil {
			if content := msg.Content; content != nil {
				if details := content.UdpChannelDetails; details != nil {
					if err := self.udpTunnel.SetUdpTunnel(details); err != nil {
						debug.Printf("Control Recive Message error: %s\n", err.Error())
						return nil, err
					}
					return self.Update()
				} else if content.Unauthorized {
					self.ControlChannel.SetExpired()
				} else if pong := content.Pong; pong != nil {
					self.lastPong = time.Now()
					if pong.ClientAddr.Compare(self.ControlChannel.Conn.Pong.ClientAddr) != 0 {
						debug.Println("client ip changed", pong.ClientAddr.String(), self.ControlChannel.Conn.Pong.ClientAddr.String())
					}
				}
			}
		}
	}
	if self.lastPong.UnixMilli() != 0 && time.Now().UnixMilli()-self.lastPong.UnixMilli() > 6_000 {
		self.lastPong = *new(time.Time)
		self.ControlChannel.SetExpired()
	}
	return nil, nil
}
