package tunnel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

type UdpTunnel struct {
	Udp4        *net.UDPConn
	Udp6        *net.UDPConn
	locker      sync.RWMutex
	Details     ChannelDetails
	LastConfirm atomic.Uint32
	LastSend    atomic.Uint32
}

type ChannelDetails struct {
	Udp         *proto.UdpChannelDetails
	AddrHistory []netip.AddrPort
}

func AssignUdpTunnel(tunUdp *UdpTunnel) error {
	// LogDebug.Println("Assign UDP Tunnel IPv4")
	udp4, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return err
	}
	tunUdp.Udp4 = udp4
	// IPv6 opcional
	// LogDebug.Println("Assign UDP Tunnel IPv6")
	if tunUdp.Udp6, err = net.ListenUDP("udp6", nil); err != nil {
		// LogDebug.Println("Cannot listen IPv6 Udp Tunnel")
		tunUdp.Udp6 = nil
		err = nil
	}

	tunUdp.Details = ChannelDetails{
		AddrHistory: []netip.AddrPort{},
		Udp:         nil,
	}

	tunUdp.LastConfirm = atomic.Uint32{}
	tunUdp.LastSend = atomic.Uint32{}
	tunUdp.LastConfirm.Store(0)
	tunUdp.LastSend.Store(0)
	return nil
}

func (udp *UdpTunnel) IsSetup() bool {
	return udp.Details.Udp != nil
}

func (udp *UdpTunnel) InvalidateSession() {
	udp.LastConfirm.Store(0)
	udp.LastSend.Store(0)
}

func now_sec() uint32 {
	return uint32(time.Now().UnixMilli()) / 1_000
}

func (udp *UdpTunnel) RequireResend() bool {
	last_confirm := udp.LastConfirm.Load()
	/* send token every 10 seconds */
	return 10 < now_sec()-last_confirm
}

func (udp *UdpTunnel) RequiresAuth() bool {
	lastConf, lastSend := udp.LastConfirm.Load(), udp.LastSend.Load()
	if lastSend < lastConf {
		return false
	}
	return 5 < now_sec()-lastSend
}

func (udp *UdpTunnel) SetUdpTunnel(details *proto.UdpChannelDetails) error {
	// LogDebug.Println("Updating Udp Tunnel")
	// udp.locker.Lock()
	if current := udp.Details.Udp; current != nil {
		if bytes.Equal(current.Token, details.Token) && current.TunnelAddr.Compare(details.TunnelAddr) == 0 {
			// udp.locker.Unlock()
			return nil
		}
		if current.TunnelAddr.Compare(details.TunnelAddr) != 0 {
			// LogDebug.Println("changed udp tunner addr")
			oldAddr := current.TunnelAddr
			udp.Details.AddrHistory = append(udp.Details.AddrHistory, oldAddr)
		}
	}
	udp.Details.Udp = new(proto.UdpChannelDetails)
	udp.Details.Udp.Token = details.Token
	udp.Details.Udp.TunnelAddr = details.TunnelAddr
	// udp.locker.Unlock()

	return udp.SendToken(details)
}

func (udp *UdpTunnel) ResendToken() (bool, error) {
	lock := udp.Details
	if lock.Udp == nil {
		return false, nil
	} else if err := udp.SendToken(lock.Udp); err != nil {
		return false, err
	}
	return true, nil
}

func (udp *UdpTunnel) SendToken(details *proto.UdpChannelDetails) error {
	// udp.locker.RLock()
	// defer udp.locker.RUnlock()
	if details.TunnelAddr.Addr().Is4() {
		if _, err := udp.Udp4.WriteToUDPAddrPort(details.Token, details.TunnelAddr); err != nil {
			return err
		}
	} else {
		if udp.Udp6 == nil {
			return fmt.Errorf("ipv6 not supported")
		}
		if _, err := udp.Udp6.WriteToUDPAddrPort(details.Token, details.TunnelAddr); err != nil {
			return err
		}
	}
	// LogDebug.Printf("send udp session token (len=%d) to %s\n", len(details.Token), details.TunnelAddr.AddrPort.String())
	udp.LastSend.Store(now_sec())
	return nil
}

func (udp *UdpTunnel) GetSock() (*net.UDPConn, *netip.AddrPort, error) {
	// udp.locker.RLock()
	// defer udp.locker.RUnlock()

	lock := udp.Details
	if lock.Udp == nil {
		// LogDebug.Println("udp tunnel not connected")
		return nil, nil, fmt.Errorf("udp tunnel not connected")
	} else if lock.Udp.TunnelAddr.Addr().Is4() {
		return udp.Udp4, &lock.Udp.TunnelAddr, nil
	} else if udp.Udp6 == nil {
		// LogDebug.Println("ipv6 not setup")
		return nil, nil, fmt.Errorf("ipv6 not setup")
	}
	return udp.Udp6, &lock.Udp.TunnelAddr, nil
}

func (Udp *UdpTunnel) Send(data []byte, Flow UdpFlow) (int, error) {
	buff := bytes.NewBuffer([]byte{})
	if err := Flow.WriteTo(buff); err != nil {
		return 0, err
	}

	socket, addr, err := Udp.GetSock()
	if err != nil {
		return 0, err
	}

	return socket.WriteToUDPAddrPort(append(data, buff.Bytes()...), *addr)
}

func (Udp *UdpTunnel) GetToken() ([]byte, error) {
	// Udp.locker.RLock()
	// defer Udp.locker.RUnlock()
	lock := Udp.Details
	if lock.Udp == nil {
		return nil, fmt.Errorf("udp tunnel not connected")
	}
	return lock.Udp.Token[:], nil
}

type UdpTunnelRxPacket struct {
	Bytes uint64
	Flow  UdpFlow
}
type UdpTunnelRx struct {
	ConfirmerdConnection bool
	ReceivedPacket       UdpTunnelRxPacket
}

func (Udp *UdpTunnel) ReceiveFrom(buff []byte) (*UdpTunnelRx, error) {
	// Udp.locker.RLock()
	// defer Udp.locker.RUnlock()

	udp, tunnelAddr, err := Udp.GetSock()
	if err != nil {
		return nil, err
	}

	// udp.SetReadDeadline(time.Now().Add(time.Second * 2))
	byteSize, remote, err := udp.ReadFromUDPAddrPort(buff)
	if err != nil {
		return nil, err
	}
	if tunnelAddr.Compare(remote) != 0 {
		lock := Udp.Details
		if !slices.ContainsFunc(lock.AddrHistory, func(a netip.AddrPort) bool {
			return a.Compare(remote) == 0
		}) {
			return nil, fmt.Errorf("got data from other source")
		}
	}
	buff = buff[:byteSize]
	token, err := Udp.GetToken()
	if err != nil {
		return nil, err
	}

	var point UdpTunnelRx
	if bytes.Equal(buff, token) {
		// LogDebug.Println("udp session confirmed")
		Udp.LastConfirm.Store(now_sec())
		point.ConfirmerdConnection = true
		return &point, nil
	} else if len(buff)+V6_LEN < byteSize {
		return nil, fmt.Errorf("receive buffer too small")
	}

	footer, footerInt, err := FromTailUdpFlow(buff)
	if err != nil {
		if footerInt == UDP_CHANNEL_ESTABLISH_ID {
			actual := hex.EncodeToString(buff)
			expected := hex.EncodeToString(token)
			return nil, fmt.Errorf("unexpected UDP establish packet, actual: %s, expected: %s", actual, expected)
		}
		return nil, fmt.Errorf("failed to extract udp footer: %s, err: %s", hex.EncodeToString(buff), err.Error())
	}
	point.ReceivedPacket = UdpTunnelRxPacket{
		uint64(byteSize - footer.Len()),
		footer,
	}
	return &point, nil
}
