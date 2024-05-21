package tunnel

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type UdpTunnel struct {
	inner *Inner
}

type Inner struct {
	Udp4        *net.UDPConn
	Udp6        *net.UDPConn
	Details     sync.RWMutex
	UdpDetails  *UdpChannelDetails
	AddrHistory []netip.Addr
	LastConfirm atomic.Uint32
	LastSend    atomic.Uint32
}

func NewUdpTunnel() (*UdpTunnel, error) {
	udp4, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}

	var udp6 *net.UDPConn
	udp6, _ = net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6unspecified, Port: 0}) // IPv6 opcional

	return &UdpTunnel{
		inner: &Inner{
			Udp4:        udp4,
			Udp6:        udp6,
			UdpDetails:  nil,
			AddrHistory: []netip.Addr{},
			LastConfirm: atomic.Uint32{},
			LastSend:    atomic.Uint32{},
		},
	}, nil
}

func (udp *UdpTunnel) IsSetup() bool {
	udp.inner.Details.RLock()
	return udp.inner.UdpDetails != nil
}

func (udp *UdpTunnel) InvalidateSession() {
	fmt.Printf("InvalidateSession %s\n", udp.inner.UdpDetails.TunnelAddr.String())
	udp.inner.LastConfirm.Store(0)
	udp.inner.LastSend.Store(0)
}

func now_sec() uint32 {
	return uint32(time.Now().UnixMilli()) / 1_000
}

func (udp *UdpTunnel) RequireResend() bool {
	last_confirm := udp.inner.LastConfirm.Load()
	/* send token every 10 seconds */
	return 10 < now_sec()-last_confirm
}

func (udp *UdpTunnel) RequiresAuth() bool {
	lastConf, lastSend := udp.inner.LastConfirm.Load(), udp.inner.LastSend.Load()
	if lastSend < lastConf {
		return false
	}
	return 5 < now_sec()-lastSend
}

func (udp *UdpTunnel) SetUdpTunnel(details UdpChannelDetails) error {
	udp.inner.Details.Lock()
	if udp.inner.UdpDetails != nil {
		if udp.inner.UdpDetails == &details {
			return nil
		}
		if details.TunnelAddr.Compare(udp.inner.UdpDetails.TunnelAddr.AddrPort) != 0 {
			udp.inner.AddrHistory = append(udp.inner.AddrHistory, udp.inner.UdpDetails.TunnelAddr.AddrPort.Addr())
		}
		udp.inner.UdpDetails = &details
	}
	return udp.SendToken(&details)
}

func (udp *UdpTunnel) ResendToken() (bool, error) {
	udp.inner.Details.Lock()
	if udp.inner.UdpDetails == nil {
		return false, nil
	}
	return true, udp.SendToken(udp.inner.UdpDetails)
}

func (ut *UdpTunnel) SendToken(details *UdpChannelDetails) error {
	var conn *net.UDPConn
	addr := details.TunnelAddr.Addr()
	if addr.Is4() {
		conn = ut.inner.Udp4
	} else {
		if ut.inner.Udp6 == nil {
			return errors.New("IPv6 not supported")
		}
		conn = ut.inner.Udp6
	}

	_, err := conn.WriteToUDP(details.Token, net.UDPAddrFromAddrPort(details.TunnelAddr.AddrPort))
	if err != nil {
		return err
	}

	fmt.Printf("send udp session token (len=%d) to %s\n", len(details.Token), details.TunnelAddr.String())
	ut.inner.LastSend.Store(now_sec())
	return nil
}

func (Udp *UdpTunnel) GetSock() (*net.UDPConn, *AddressPort, error) {
	Udp.inner.Details.RLock()
	if Udp.inner.UdpDetails == nil {
		return nil, nil, fmt.Errorf("udp tunnel not connected")
	} else if Udp.inner.UdpDetails.TunnelAddr.Addr().Is4() {
		return Udp.inner.Udp4, &Udp.inner.UdpDetails.TunnelAddr, nil
	} else if Udp.inner.Udp6 == nil {
		return nil, nil, fmt.Errorf("ipv6 not setup")
	}
	return Udp.inner.Udp6, &Udp.inner.UdpDetails.TunnelAddr, nil
}

func (Udp *UdpTunnel) Send(data []byte, Flow UdpFlow) error {
	err := Flow.WriteTo(bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	conn, addr, err := Udp.GetSock()
	if err != nil {
		return err
	}
	_, err = conn.WriteTo(data, net.UDPAddrFromAddrPort(addr.AddrPort))
	return err
}

func (Udp *UdpTunnel) GetToken() ([]byte, error) {
	Udp.inner.Details.RLock()
	if Udp.inner.UdpDetails == nil {
		return nil, fmt.Errorf("udp tunnel not connected")
	}
	return Udp.inner.UdpDetails.Token, nil
}

type UdpTunnelRx struct {
	ConfirmerdConnection bool
	ReceivedPacket       *struct {
		Bytes uint64
		Flow  UdpFlow
	}
}

func (Udp *UdpTunnel) ReceiveFrom(buff []byte) (*UdpTunnelRx, error) {
	conn, addr, err := Udp.GetSock()
	if err != nil {
		return nil, err
	}
	sizeRead, remote, err := conn.ReadFrom(buff)
	if err != nil {
		return nil, err
	}
	if remote.String() != addr.Addr().String() {
		Udp.inner.Details.RLock()
		found := false
		for _, addr := range Udp.inner.AddrHistory {
			if addr.String() == remote.String() {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("got data from another source")
		}
	}
	token, err := Udp.GetToken()
	if err != nil {
		return nil, err
	} else if slices.Compare(token, buff[sizeRead:]) == 0 {
		Udp.inner.LastConfirm.Store(now_sec())
		return &UdpTunnelRx{ConfirmerdConnection: true}, nil
	}
	udpFlow, footer, err := FromTailUdpFlow(buff[sizeRead:])
	if err != nil {
		return nil, err
	} else if udpFlow == nil {
		if footer == UDP_CHANNEL_ESTABLISH_ID {
			act, exp := hex.EncodeToString(buff[sizeRead:]), hex.EncodeToString(token)
			return nil, fmt.Errorf("unexpected UDP establish packet, actual: %q, expected: %q", act, exp)
		}
	}
	return &UdpTunnelRx{
		ReceivedPacket: &struct{Bytes uint64; Flow UdpFlow}{
			Bytes: uint64(sizeRead) - uint64(udpFlow.Len()),
			Flow: *udpFlow,
		},
	}, nil
}
