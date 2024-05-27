package runner

import (
	"fmt"
	"net/netip"
	"sync"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/network"
)

type TunnelEntry struct {
	PubAddress       string
	MatchIP          network.MatchIP
	PortType         api.PortProto
	FromPort, ToPort uint16
	LocalStartAdress netip.AddrPort
}

type LocalLookup struct {
	AdreessLock sync.Mutex
	Adreess     []TunnelEntry
}

func (look *LocalLookup) Lookup(IP netip.Addr, Port uint16, Proto api.PortProto) *network.AddressValue[netip.AddrPort] {
	// look.AdreessLock.Lock()
	// defer look.AdreessLock.Unlock()
	for _, tunnel := range look.Adreess {
		if tunnel.PortType != Proto && tunnel.PortType != "both" {
			continue
		} else if !tunnel.MatchIP.Matches(IP) {
			continue
		} else if  tunnel.FromPort <= Port && Port < tunnel.ToPort {
			return &network.AddressValue[netip.AddrPort]{
				Value: tunnel.LocalStartAdress,
				FromPort: tunnel.FromPort,
				ToPort: tunnel.ToPort,
			}
		}
	}
	return nil
}

func (look *LocalLookup) Update(tunnels []api.AgentTunnel) {
	entries := []TunnelEntry{}
	for _, tunnel := range tunnels {
		tun := TunnelEntry{
			PortType: api.PortProto(tunnel.Proto),
			FromPort: tunnel.Port.From,
			ToPort: tunnel.Port.To,
			LocalStartAdress: netip.AddrPortFrom(tunnel.LocalIp, tunnel.LocalPort),
			MatchIP: network.MatchIP{IPNumber: uint64(tunnel.IpNum)},
		}
		if tunnel.RegionNum != 0 {
			tun.MatchIP.RegionID = new(uint16)
			*tun.MatchIP.RegionID = tunnel.RegionNum
		}
		entries = append(entries, tun)
	}
	look.AdreessLock.Lock()
	defer look.AdreessLock.Unlock()
	look.Adreess = entries
}

func Autorun(Api api.Api) error {
	lookup := LocalLookup{Adreess: []TunnelEntry{}, AdreessLock: sync.Mutex{}}
	tuns, err := Api.AgentInfo()
	if err != nil {
		return err
	}
	lookup.Update(tuns.Tunnels)
	for _, tun := range tuns.Tunnels {
		src, dst := tun.SourceString(), tun.DestString()
		if tun.Disabled != nil {
			fmt.Printf("%s -> %s (Disabled)\n", src, dst)
		} else if tun.TunnelType != "" {
			fmt.Printf("%s -> %s (%s)\n", src, dst, tun.TunnelType)
		} else {
			fmt.Printf("%s -> %s (Proto: %s, Port Count %d)\n", src, dst, tun.Proto, tun.Port.To - tun.Port.From)
		}
	}
	var runner TunnelRunner
	errorCount := 0
	for {
		runner, err = NewTunnelRunner(Api, &lookup)
		if err == nil {
			break
		} else if errorCount++; errorCount > 5 {
			return err
		}
		<-time.After(time.Second*2)
	}
	runing := runner.Run()
	go func(){
		for runner.KeepRunning.Load() {
			if tuns, err = Api.AgentInfo(); err != nil {
				<-time.After(time.Second*3)
				continue
			}
			lookup.Update(tuns.Tunnels)
			<-time.After(time.Second*3)
		}
	}()
	defer runner.KeepRunning.Store(false)
	return <- runing
}
