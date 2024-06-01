package network

import (
	"encoding/binary"
	"net"
	"net/netip"
)

func shuffle(v uint32) uint32 {
	v = ((v >> 16) ^ v) * 0x45d9f3
	v = ((v >> 16) ^ v) * 0x45d9f3
	v = (v >> 16) ^ v
	return v
}

func asLocalMasked(ip uint32) uint32 {
	ip = shuffle(ip) & 0x00FFFFFF
	if ip == 0 {
		ip = 1
	}
	return ip | 0x7F000000
}

func mapToLocalIP4(ip net.IP) netip.Addr {
	var ipUint32 uint32
	if ip.To4() != nil { // Check if it's already IPv4
		ipUint32 = binary.BigEndian.Uint32(ip.To4())
	} else { // Handle IPv6
		bytes := ip.To16() // Convert to IPv6 bytes
		ipUint32 = shuffle(binary.BigEndian.Uint32(bytes[0:4])) ^
			shuffle(binary.BigEndian.Uint32(bytes[4:8])) ^
			shuffle(binary.BigEndian.Uint32(bytes[8:12])) ^
			shuffle(binary.BigEndian.Uint32(bytes[12:16]))
	}

	return netip.AddrFrom4([4]byte{
		byte(asLocalMasked(ipUint32)>>24),
		byte(asLocalMasked(ipUint32)>>16),
		byte(asLocalMasked(ipUint32)>>8),
		byte(asLocalMasked(ipUint32)),
	})
}

func TcpSocket(SpecialLan bool, Peer, Host netip.AddrPort) (*net.TCPConn, error) {
	isLoopback := Host.Addr().IsLoopback()
	if isLoopback && SpecialLan {
		stream, err := net.DialTCP("tcp", net.TCPAddrFromAddrPort(netip.AddrPortFrom(mapToLocalIP4(Peer.Addr().AsSlice()), 0)), net.TCPAddrFromAddrPort(Host))
		if err == nil {
			return stream, nil
		}
		// logDebug.Printf("Failed to establish connection using special lan %s for flow %s -> %s\n", local_ip, Peer.String(), Host.String())
	}
	// logDebug.Printf("Failed to bind connection to special local address to support IP based banning")
	stream, err := net.DialTCP("tcp", nil, net.TCPAddrFromAddrPort(Host))
	if err != nil {
		// logDebug.Printf("Failed to establish connection for flow %s -> %s. Is your server running? %q", Peer.String(), Host.String(), err.Error())
		return nil, err
	}
	return stream, nil
}

func UdpSocket(SpecialLan bool, Peer, Host netip.AddrPort) (*net.UDPConn, error) {
	isLoopback := Host.Addr().IsLoopback()
	if isLoopback && SpecialLan {
		local_ip := mapToLocalIP4(Peer.Addr().AsSlice());
		local_port := 40000 + (Peer.Port() % 24000);
		stream, err := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(netip.AddrPortFrom(local_ip, local_port)))
		if err != nil {
			stream, err = net.ListenUDP("udp4", net.UDPAddrFromAddrPort(netip.AddrPortFrom(local_ip, 0)))
			if err != nil {
				stream, err = net.ListenUDP("udp4", nil)
			}
		}
		return stream, err
	}
	return net.ListenUDP("udp", nil)
}