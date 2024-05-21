package tunnel

import (
	"fmt"
	"net"
	"sync/atomic"
)

type ActiveClient struct {
	ClaimInstructions ClaimInstructions
	Dial              net.Conn
}

type TunnelRunner struct {
	Tunnel      SimplesTunnel
	KeepRunning atomic.Bool
	TcpClients  map[string]ActiveClient
	UdpCleints  map[string]ActiveClient
}

func (tun *TunnelRunner) Run() error {
	channel := make(chan error)
	// Assign ner clients to tunnel
	tun.TcpClients = map[string]ActiveClient{}
	tun.UdpCleints = map[string]ActiveClient{}

	// Setup Tunnel
	if err := tun.Tunnel.Setup(); err != nil {
		return err
	}
	fmt.Println("Success Tunnel setup")

	go func() {
		for tun.KeepRunning.Load() {
			newClient, err := tun.Tunnel.Update()
			if err != nil {
				channel <- err
				return
			} else if newClient == nil {
				continue
			}
			fmt.Printf("%#v\n", newClient)
		}
	}()
	return <-channel
}
