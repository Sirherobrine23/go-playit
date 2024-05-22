package tunnel

import (
	"net/netip"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

type TunnelRunner struct {
	Lookup      AddressLookup[netip.AddrPort]
	Tunnel      SimplesTunnel
	KeepRunning atomic.Bool
}

func (tun *TunnelRunner) UseSpecialLan(set bool) {
	panic("no implemented UseSpecialLan")
}

func (tun *TunnelRunner) Run() error {
	channel := make(chan error)
	// udp := tun.Tunnel.UdpTunnel

	// Setup Tunnel
	if err := tun.Tunnel.Setup(); err != nil {
		return err
	}
	LogDebug.Println("Success Tunnel setup")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		LogDebug.Println("Cleaning process")
		tun.KeepRunning.Store(false)
		tun.Tunnel.ControlChannel.Conn.Udp.Close()
		os.Exit(1)
	}()

	// TCP Clients
	go func() {
		lastControlUpdate := time.Now().UnixMilli()
		defer os.Exit(2)
		for tun.KeepRunning.Load() {
			now := time.Now().UnixMilli()
			if 30_000 < now-lastControlUpdate {
				lastControlUpdate = now
				LogDebug.Println("Reloading control addr")
				if _, err := tun.Tunnel.ReloadControlAddr(); err != nil {
					LogDebug.Println("failed to reload control addr")
					LogDebug.Println(err)
					return
				}
			}

			newClient, err := tun.Tunnel.Update()
			if err != nil {
				LogDebug.Println(err.Error())
				channel <- err
				return
			} else if newClient == nil {
				continue
			}
			LogDebug.Println("tcp client")
			LogDebug.Printf("%#v\n", newClient)
			LogDebug.Panic(newClient)
		}
	}()

	// go func() {
	// 	buffer := make([]byte, 2048)
	// 	had_success := false
	// 	for tun.KeepRunning.Load() {
	// 		LogDebug.Println("rec udp tun")
	// 		rx, err := udp.ReceiveFrom(buffer)
	// 		if err != nil {
	// 			LogDebug.Println(err)
	// 			if had_success {
	// 				LogDebug.Panicln("got error")
	// 			}
	// 			time.Sleep(time.Second)
	// 			continue
	// 		}
	// 		LogDebug.Println("success")
	// 		had_success = true
	// 		if rx.ConfirmerdConnection {
	// 			continue
	// 		}
	// 		LogDebug.Printf("%#v\n", rx.ReceivedPacket)
	// 	}
	// }()
	return <-channel
}
