package runner

import (
	"fmt"
	"io"
	"net"
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
	udp_clients := network.NewUdpClients(*tunnel.UdpTunnel(), Lookup)
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

func (self *TunnelRunner) Run() chan error {
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
			fmt.Println("Reciving new connection")
			if new_client := tunnel.Update(); new_client != nil {
				fmt.Println("New TCP Client")
				found := self.Lookup.Lookup(new_client.ConnectAddr.Addr(), new_client.ConnectAddr.Port(), api.PortProto("tcp"))
				if found == nil {
					fmt.Println("could not find local address for connection")
					continue
				}
				go func() {
					var (
						tunnel_conn *network.TcpClient
						local_conn  *net.TCPConn
						err         error
					)

					if tunnel_conn, err = self.TcpClients.Connect(*new_client); err != nil {
						return
					}
					defer tunnel_conn.Stream.Close()
					defer tunnel_conn.Dropper.Drop()

					if local_conn, err = network.TcpSocket(self.TcpClients.UseSpecialLAN, new_client.PeerAddr, netip.AddrPortFrom(found.Value.Addr(), (new_client.ConnectAddr.Port()-found.FromPort)+found.Value.Port())); err != nil {
						return
					}
					defer local_conn.Close()
					done := make(chan struct{})
					defer close(done)
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

	// go func(){
	// 	udp := tunnel.UdpTunnel()
	// 	for self.KeepRunning.Load() {
	// 		buffer := make([]byte, 2048)
	// 		fmt.Println("udp rec")
	// 		rx, err := udp.ReceiveFrom(buffer)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			time.Sleep(time.Second)
	// 			continue
	// 		}
	// 		if rx.ConfirmerdConnection {
	// 			continue
	// 		}
	// 		d,_:=json.MarshalIndent(rx, "", "  ")
	// 		fmt.Printf("rx: %s\n", string(d))
	// 		bytes, flow := rx.ReceivedPacket.Bytes, rx.ReceivedPacket.Flow
	// 		if err := self.UdpClients.ForwardPacket(flow, buffer[:bytes]); err != nil {
	// 			panic(err)
	// 		}
	// 	}
	// }()
	return end
}
