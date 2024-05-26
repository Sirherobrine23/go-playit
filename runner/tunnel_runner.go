package runner

import (
	"fmt"
	"io"
	"net/netip"
	"sync/atomic"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/network"
	"sirherobrine23.org/playit-cloud/go-playit/tunnel"
)

type TunnelRunner struct {
	Lookup      network.AddressLookup[netip.AddrPort]
	Tunnel      tunnel.SimpleTunnel
	UdpClients  network.UdpClients
	TcpClients  network.TcpClients
	KeepRunning atomic.Bool
}

func NewTunnelRunner(Api api.Api, Lookup network.AddressLookup[netip.AddrPort]) (TunnelRunner, error) {
	tunnel := tunnel.NewSimpleTunnel(Api)
	if err := tunnel.Setup(); err != nil {
		return TunnelRunner{}, err
	}
	udp_clients := network.NewUdpClients(tunnel.UdpTunnel(), Lookup)
	var keep atomic.Bool
	keep.Store(true)
	return TunnelRunner{
		Lookup:      Lookup,
		Tunnel:      tunnel,
		UdpClients:  udp_clients,
		TcpClients:  network.NewTcpClients(),
		KeepRunning: keep,
	}, nil
}

func (self *TunnelRunner) SetSpecialLan(setUse bool) {
	self.TcpClients.UseSpecialLAN = setUse
	self.UdpClients.UseSpecialLan = setUse
}

func (self *TunnelRunner) Run() {
	end := make(chan error)
	tunnel := self.Tunnel
	go func() {
		lastControlUpdate := time.Now().UnixMilli()
		for self.KeepRunning.Load() {
			now := time.Now().UnixMilli()
			if 30_000 < time.Now().UnixMilli()-lastControlUpdate {
				lastControlUpdate = now
				if _, err := tunnel.ReloadControlAddr(); err != nil {
				}
			}
			fmt.Println("Reload")
			if new_client := tunnel.Update(); new_client != nil {
				fmt.Println("New TCP Client")
				clients := self.TcpClients
				found := self.Lookup.Lookup(new_client.ConnectAddr.Addr(), new_client.ConnectAddr.Port(), api.PortProto("tcp"))
				if found == nil {
					continue
				}
				local_addr := netip.AddrPortFrom(found.Value.Addr(), (new_client.ConnectAddr.Port()-found.FromPort)+found.Value.Port())
				go func() {
					peerAddr := new_client.PeerAddr
					tunnel_conn, err := clients.Connect(*new_client)
					if err != nil {
						return
					}
					defer tunnel_conn.Stream.Close()
					local_conn, err := network.TcpSocket(self.TcpClients.UseSpecialLAN, peerAddr, local_addr)
					if err != nil {

					}
					defer local_conn.Close()
					done := make(chan struct{})
					go func() {
						io.Copy(&tunnel_conn.Stream, local_conn)
						done <- struct{}{}
					}()

					go func() {
						io.Copy(local_conn, &tunnel_conn.Stream)
						done <- struct{}{}
					}()
					<-done
					<-done
				}()
			}
		}
	}()

	// udp_clients := self.UdpClients
	// go func(){
	// 	buffer := make([]byte, 2048)
	// 	// had_success := false
	// 	for self.KeepRunning.Load() {
	// 		rx, err := udp.ReceiveFrom(buffer)
	// 		fmt.Println(rx)
	// 		if err != nil {
	// 			time.Sleep(time.Second)
	// 			continue
	// 		}
	// 		if rx.ConfirmerdConnection {
	// 			continue
	// 		}
	// 		bytes, flow := rx.ReceivedPacket.Bytes, rx.ReceivedPacket.Flow
	// 		if err := udp_clients.ForwardPacket(flow, buffer[:bytes]); err != nil {
	// 			panic(err)
	// 		}
	// 	}
	// }()
	<-end
}
