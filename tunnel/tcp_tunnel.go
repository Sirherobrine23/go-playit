package tunnel

import "net"

type TcpTunnel struct {
	ClaimInstructions ClaimInstructions
}
func (tcp *TcpTunnel) Connect() (*net.TCPConn, error) {
	stream, err := net.DialTCP("tcp", nil, net.TCPAddrFromAddrPort(tcp.ClaimInstructions.Address.AddrPort))
	if err != nil {
		LogDebug.Printf("%q: Failed to establish connection to tunnel server\n", tcp.ClaimInstructions.Address.AddrPort.String())
		return nil, err
	}
	if _, err := stream.Write(tcp.ClaimInstructions.Token); err != nil {
		stream.Close()
		return nil, err
	}
	res := make([]byte, 8)
	if _, err := stream.Read(res); err != nil {
		stream.Close()
		return nil, err
	}
	LogDebug.Printf("%+v\n", res)
	return stream, nil
}