package tunnel

import (
	"fmt"
	"net"

	"sirherobrine23.org/playit-cloud/go-playit/proto"
)

type TcpTunnel struct {
	ClaimInstruction proto.ClaimInstructions
}

func (tcpTunnel *TcpTunnel) Connect() (*net.TCPConn, error) {
	conn, err := net.DialTCP("tcp", nil, net.TCPAddrFromAddrPort(tcpTunnel.ClaimInstruction.Address))
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}
	_, err = conn.Write(tcpTunnel.ClaimInstruction.Token)
	if err != nil {
		conn.Close()
		return nil, err
	}
	buff := make([]byte, 8)
	size, err := conn.Read(buff)
	if err != nil {
		conn.Close()
		return nil, err
	} else if size != 8 {
		conn.Close()
		return nil, fmt.Errorf("invalid response reader size")
	}
	return conn, nil
}