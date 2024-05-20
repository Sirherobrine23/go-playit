package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
)

type ActiveClient struct {
	ClaimInstructions ClaimInstructions
	Dial              net.Conn
}

type TunnelRunner struct {
	Tunnel     SimplesTunnel
	TcpClients map[string]ActiveClient
	UdpCleints map[string]ActiveClient
}

func (tun *TunnelRunner) Run() error {
	// Assign ner clients to tunnel
	tun.TcpClients = map[string]ActiveClient{}
	tun.UdpCleints = map[string]ActiveClient{}

	// Setup Tunnel
	if err := tun.Tunnel.Setup(); err != nil {
		return err
	}

	for {
		newClient := tun.Tunnel.Update()
		if newClient == nil {
			continue
		}
		fmt.Println("New client")
		dial, err := net.Dial("tcp", newClient.ClaimInstructions.Address.String())
		if err != nil {
			return err
		}
		fmt.Println("Success connect")
		if _, err := dial.Write(newClient.ClaimInstructions.Token[:]); err != nil {
			return err
		}
		recBuff := make([]byte, 8)
		size, err := dial.Read(recBuff)
		if err != nil {
			return err
		} else if size != 8 {
			return fmt.Errorf("Panic - Caravan Palace")
		}
		fmt.Println("Success auth")
		tun.TcpClients[newClient.ConnectAddr.String()] = ActiveClient{
			ClaimInstructions: newClient.ClaimInstructions,
			Dial:              dial,
		}
		req, err := http.ReadRequest(bufio.NewReader(dial))
		if err != nil {
			return err
		}
		req.Response.Write(bytes.NewBuffer([]byte("Google")))
		req.Response.Body.Close()
	}

	return nil
}
