package network

import (
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"sync/atomic"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/tunnel"
)

type UdpClient struct {
	clientKey      ClientKey
	sendFlow       tunnel.UdpFlow
	udpTunnel      tunnel.UdpTunnel
	localUdp       *net.UDPConn
	localStartAddr netip.AddrPort
	tunnelFromPort uint16
	tunnelToPort   uint16
	udpClients     map[ClientKey]UdpClient
	lastActivity   atomic.Uint32
}

func (self *UdpClient) SendLocal(dstPort uint16, data []byte) error {
	portOffset := dstPort - self.tunnelFromPort
	self.lastActivity.Store(uint32(time.Now().UnixMilli() / 1_000))
	if portOffset == 0 {
		_, err := self.localUdp.WriteToUDP(data, net.UDPAddrFromAddrPort(self.localStartAddr))
		return err
	}
	_, err := self.localUdp.WriteToUDP(data, net.UDPAddrFromAddrPort(netip.AddrPortFrom(self.localStartAddr.Addr(), self.localStartAddr.Port()+portOffset)))
	return err
}

type HostToTunnelForwarder struct{ UdpClient }

func (self *HostToTunnelForwarder) Run() {
	buffer := make([]byte, 2048)
	for {
		buffer = make([]byte, 2048)
		self.localUdp.SetReadDeadline(time.Now().Add(time.Second * 30))
		size, source, err := self.localUdp.ReadFromUDPAddrPort(buffer)
		if err != nil {
			debug.Println(err)
			break
		} else if source.Addr().Compare(self.localStartAddr.Addr()) != 0 {
			// "dropping packet from different unexpected source"
			continue
		}

		portCount := self.tunnelToPort - self.tunnelFromPort
		localFrom := self.localStartAddr.Port()
		localTo := localFrom + portCount
		if source.Port() < localFrom || localTo <= source.Port() {
			// "dropping packet outside of expected port range"
			continue
		}
		buffer = buffer[:size]
		portOffset := source.Port() - localFrom
		flow := self.sendFlow.WithSrcPort(self.tunnelFromPort + portOffset)
		if _, err = self.udpTunnel.Send(buffer, flow); err != nil {
			// "failed to send packet to through tunnel"
		}
	}

	if _, is := self.UdpClient.udpClients[self.clientKey]; is {
		// if !reflect.DeepEqual(v, self) {} else {}
		delete(self.UdpClient.udpClients, self.clientKey)
	}
}

type ClientKey struct {
	ClientAddr, TunnelAddr netip.AddrPort
}

type UdpClients struct {
	udpTunnel        tunnel.UdpTunnel
	lookup           AddressLookup[netip.AddrPort]
	udpClients       map[ClientKey]UdpClient
	UseSpecialLan    bool
}

func NewUdpClients(Tunnel tunnel.UdpTunnel, Lookup AddressLookup[netip.AddrPort]) UdpClients {
	return UdpClients{
		udpTunnel:        Tunnel,
		lookup:           Lookup,
		udpClients:       make(map[ClientKey]UdpClient),
		UseSpecialLan:    true,
	}
}

func (self *UdpClients) ClientCount() int {
	return len(self.udpClients)
}

func (self *UdpClients) ForwardPacket(Flow tunnel.UdpFlow, data []byte) error {
	flowDst := Flow.Dst()
	found := self.lookup.Lookup(flowDst.Addr(), flowDst.Port(), api.PortProto("udp"))
	if found == nil {
		return fmt.Errorf("could not find tunnel")
	}

	key := ClientKey{ClientAddr: Flow.Src(), TunnelAddr: netip.AddrPortFrom(flowDst.Addr(), found.FromPort)}
	for kkey, client := range self.udpClients {
		if reflect.DeepEqual(kkey, key) {
			return client.SendLocal(flowDst.Port(), data)
		}
	}

	client, err := func() (*UdpClient, error) {
		for kkey, client := range self.udpClients {
			if reflect.DeepEqual(kkey, key) {
				return &client, nil
			}
		}
		localAddr := found.Value
		var sendFlow tunnel.UdpFlow
		var clientAddr netip.AddrPort
		if Flow.IPSrc.Addr().Is4() {
			clientAddr = netip.AddrPortFrom(Flow.IPSrc.Addr(), Flow.IPSrc.Port())
			sendFlow = tunnel.UdpFlow{
				IPSrc: netip.AddrPortFrom(Flow.IPDst.Addr(), found.FromPort),
				IPDst: Flow.Src(),
			}
		} else {
			clientAddr = netip.AddrPortFrom(Flow.IPSrc.Addr(), Flow.IPSrc.Port())
			sendFlow = tunnel.UdpFlow{
				IPSrc: netip.AddrPortFrom(Flow.IPDst.Addr(), found.FromPort),
				IPDst: Flow.Src(),
				Flow: sendFlow.Flow,
			}
		}

		usock, err := UdpSocket(self.UseSpecialLan, clientAddr, localAddr)
		if err != nil {
			return nil, err
		}
		client := UdpClient{
			clientKey:      key,
			sendFlow:       sendFlow,
			localUdp:       usock,
			udpTunnel:      self.udpTunnel,
			localStartAddr: localAddr,
			tunnelFromPort: found.FromPort,
			tunnelToPort:   found.ToPort,
			udpClients:     self.udpClients,
			lastActivity:   atomic.Uint32{},
		}

		self.udpClients[key] = client
		go (&HostToTunnelForwarder{client}).Run()
		return &client, nil
	}()
	if err != nil {
		return err
	}
	return client.SendLocal(flowDst.Port(), data)
}
