package runner

import (
	"io"
	"net"
	"net/netip"
	"sync/atomic"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/logfile"
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
					<-time.After(time.Second * 3)
					continue
				}
			}
			new_client, err := tunnel.Update()
			if err != nil {
				debug.Printf("Error recived: %s\n", err.Error())
				<-time.After(time.Second)
				continue
			} else if new_client == nil {
				<-time.After(time.Second)
				continue
			}
			debug.Println("New TCP Client")
			var found *network.AddressValue[netip.AddrPort]
			if found = self.Lookup.Lookup(new_client.ConnectAddr.Addr(), new_client.ConnectAddr.Port(), api.PortProto("tcp")); found == nil {
				debug.Println("could not find local address for connection")
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
				if tunnel_conn.Stream != nil {
					defer tunnel_conn.Stream.Close()
				}

				if local_conn, err = network.TcpSocket(self.TcpClients.UseSpecialLAN, new_client.PeerAddr, netip.AddrPortFrom(found.Value.Addr(), (new_client.ConnectAddr.Port()-found.FromPort)+found.Value.Port())); err != nil {
					debug.Println(err)
					return
				}
				defer local_conn.Close()
				done := make(chan struct{})
				defer close(done)
				go func() {
					io.Copy(tunnel_conn.Stream, local_conn)
					done <- struct{}{}
				}()
				go func() {
					io.Copy(local_conn, tunnel_conn.Stream)
					done <- struct{}{}
				}()
				<-done
				<-done
			}()
		}
		end <- nil
	}()

	go func() {
		udp := tunnel.UdpTunnel()
		for self.KeepRunning.Load() {
			buffer, rx, err := udp.ReceiveFrom()
			if err != nil {
				// if et, is := err.(net.Error); is && !et.Timeout() {
				debug.Printf("UdpTunnel Error: %s\n", err.Error())
				// }
				time.Sleep(time.Second)
				continue
			}
			debug.Printf("UdpTunnel: %s\n", logfile.JSONString(rx))
			if rx.ConfirmerdConnection {
				continue
			} else if err := self.UdpClients.ForwardPacket(rx.ReceivedPacket.Flow, buffer); err != nil {
				debug.Println(err)
				panic(err)
			}
		}
	}()
	return end
}
