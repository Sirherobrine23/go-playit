package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/google/uuid"
)

type AssignedDefaultCreate struct {
	Ip   net.IP  `json:"local_ip"`
	Port *uint16 `json:"local_port,omitempty"`
}

type AssignedAgentCreate struct {
	ID   uuid.UUID `json:"agent_id"`
	Ip   net.IP    `json:"local_ip"`
	Port *uint16   `json:"local_port,omitempty"`
}

type AssignedManagedCreate struct {
	ID *uuid.UUID `json:"agent_id,omitempty"`
}

type TunnelOriginCreate struct {
	Type  string `json:"type"` // Agent type: default, agent or managed
	Agent any    `json:"data"` // Assingned agent
}

func (Origin *TunnelOriginCreate) Check() error {
	if _, isDefault := Origin.Agent.(AssignedDefaultCreate); Origin.Type == "default" && !isDefault {
		return fmt.Errorf("current agent is default, so Agent don't is AssignedDefaultCreate")
	} else if _, isAgentCreate := Origin.Agent.(AssignedAgentCreate); Origin.Type == "agent" && !isAgentCreate {
		return fmt.Errorf("current agent is agent, so Agent don't is AssignedAgentCreate")
	} else if _, isManagedCreate := Origin.Agent.(AssignedManagedCreate); Origin.Type == "managed" && !isManagedCreate {
		return fmt.Errorf("current managed is default, so Agent don't is AssignedManagedCreate")
	}
	return nil
}

type UseAllocDedicatedIp struct {
	IpHost string  `json:"ip_hostname"`
	Port   *uint16 `json:"port,omitempty"`
}

type UseAllocPortAlloc struct {
	ID uuid.UUID `json:"alloc_id"`
}

type UseRegion struct {
	Region string `json:"region"`
}
/**
"status": "allocated",
"data": {
	"assigned_domain": "going-scales.gl.at.ply.gg",
	"assigned_srv": null,
	"assignment": {
		"type": "shared-ip"
	},
	"id": "f667b538-0294-4817-9332-5cba5e94d79e",
	"ip_hostname": "19.ip.gl.ply.gg",
	"ip_type": "both",
	"port_end": 49913,
	"port_start": 49912,
	"region": "global",
	"static_ip4": "147.185.221.19",
	"tunnel_ip": "2602:fbaf:0:1::13"
}
*/
type TunnelCreateUseAllocation struct {
	Status string `json:"status"`  // For tunnel list
	Type   string `json:"type"`    // "dedicated-ip", "port-allocation" or "region"
	Data   any    `json:"details"` // UseAllocDedicatedIp, UseAllocPortAlloc, UseRegion
}

func (Alloc *TunnelCreateUseAllocation) Check() error {
	if _, isDedicatedIp := Alloc.Data.(UseAllocDedicatedIp); isDedicatedIp {
		Alloc.Type = "dedicated-ip"
		return nil
	} else if _, isPortAlloc := Alloc.Data.(UseAllocPortAlloc); isPortAlloc {
		Alloc.Type = "port-allocation"
		return nil
	} else if Region, isRegion := Alloc.Data.(UseRegion); isRegion {
		if slices.Contains(Regions, Region.Region) {
			Alloc.Type = "region"
			return nil
		}
		return fmt.Errorf("set valid region")
	}
	return fmt.Errorf("invalid allocation type")
}

type Tunnel struct {
	ID         *uuid.UUID                 `json:"tunnel_id,omitempty"`   // Tunnel UUID
	Name       string                     `json:"name,omitempty"`        // Tunnel name
	TunnelType string                     `json:"tunnel_type,omitempty"` // Tunnel type from TunnelType const's
	PortType   string                     `json:"port_type"`             // tcp, udp or both
	PortCount  uint16                     `json:"port_count"`            // Port count to assign to connect
	Origin     TunnelOriginCreate         `json:"origin"`
	Enabled    bool                       `json:"enabled"`
	Alloc      *TunnelCreateUseAllocation `json:"alloc,omitempty"`
	Firewall   *uuid.UUID                 `json:"firewall_id,omitempty"` // Firewall ID
}

func (w *Api) CreateTunnel(tun Tunnel) error {
	var err error
	if err = tun.Alloc.Check(); err != nil {
		return err
	} else if err = tun.Origin.Check(); err != nil {
		return err
	} else if len(tun.TunnelType) > 0 && !slices.Contains(TunnelType, tun.TunnelType) {
		return fmt.Errorf("invalid tunnel type")
	}

	// encode json body
	body, err := json.MarshalIndent(&tun, "", "  ")
	if err != nil {
		return err
	}

	var tunnelId struct {
		ID uuid.UUID `json:"id"`
	}
	if _, err = w.requestToApi("/tunnels/create", bytes.NewReader(body), &tunnelId, nil); err != nil {
		return err
	}
	tun.ID = &tunnelId.ID

	info, err := w.AgentInfo()
	if err != nil {
		return err
	}

	for {
		tuns, err := w.ListTunnels(tun.ID, &info.ID)
		if err != nil {
			return err
		}
		if tuns.Tunnels[0].Alloc.Status == "pending" {
			time.Sleep(time.Second * 2)
			continue
		}
		break
	}

	return nil
}

func (w *Api) DeleteTunnel(TunnelID *uuid.UUID) error {
	if TunnelID == nil {
		return nil
	}
	body, err := json.Marshal(struct {
		TunnelID uuid.UUID `json:"tunnel_id"`
	}{*TunnelID})
	if err != nil {
		return err
	}

	_, err = w.requestToApi("/tunnels/delete", bytes.NewReader(body), nil, nil)
	return err
}

type AccountTunnel struct {
	ID         uuid.UUID          `json:"id"`
	TunnelType string             `json:"tunnel_type"`
	CreatedAt  time.Time          `json:"created_at"`
	Name       string             `json:"name"`
	PortType   string             `json:"port_type"`
	PortCount  int32              `json:"port_count"`
	Alloc      TunnelCreateUseAllocation                `json:"alloc"`
	Origin     TunnelOriginCreate `json:"origin"`
	Domain     *struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		IsExternal bool      `json:"is_external"`
		Parent     string    `json:"parent"`
		Source     string    `json:"source"`
	} `json:"domain"`
	FirewallID string `json:"firewall_id"`
	Ratelimit  struct {
		BytesSecs   uint64 `json:"bytes_per_second"`
		PacketsSecs uint64 `json:"packets_per_second"`
	} `json:"ratelimit"`
	Active         bool   `json:"active"`
	DisabledReason string `json:"disabled_reason"`
	Region         string `json:"region"`
	ExpireNotice   *struct {
		Disable time.Time `json:"disable_at"`
		Remove  time.Time `json:"remove_at"`
	} `json:"expire_notice"`
}

type AlloctedPorts struct {
	Allowed uint16 `json:"allowed"`
	Claimed uint16 `json:"claimed"`
	Desired uint16 `json:"desired"`
}

type AccountTunnels struct {
	Tcp     AlloctedPorts   `json:"tcp_alloc"`
	Udp     AlloctedPorts   `json:"udp_alloc"`
	Tunnels []AccountTunnel `json:"tunnels"`
}

func (w *Api) ListTunnels(TunnelID, AgentID *uuid.UUID) (*AccountTunnels, error) {
	type TunList struct {
		TunnelID *uuid.UUID `json:"tunnel_id,omitempty"`
		AgentID  *uuid.UUID `json:"agent_id,omitempty"`
	}
	body, err := json.Marshal(TunList{TunnelID, AgentID})
	if err != nil {
		return nil, err
	}

	var Tuns AccountTunnels
	if _, err := w.requestToApi("/tunnels/list", bytes.NewBuffer(body), &Tuns, nil); err != nil {
		return nil, err
	}
	return &Tuns, nil
}
