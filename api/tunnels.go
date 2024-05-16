package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"slices"

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
	Type  string `json:"type"` // Agent type: default, agent or maneged
	Agent any    `json:"data"` // Assingned agent
}

func (Origin *TunnelOriginCreate) Check() error {
	if _, isDefault := Origin.Agent.(AssignedDefaultCreate); Origin.Type == "default" && !isDefault {
		return fmt.Errorf("current agent is default, so Agent don't is AssignedDefaultCreate")
	} else if _, isAgentCreate := Origin.Agent.(AssignedAgentCreate); Origin.Type == "agent" && !isAgentCreate {
		return fmt.Errorf("current agent is agent, so Agent don't is AssignedAgentCreate")
	} else if _, isManagedCreate := Origin.Agent.(AssignedManagedCreate); Origin.Type == "maneged" && !isManagedCreate {
		return fmt.Errorf("current maneged is default, so Agent don't is AssignedManagedCreate")
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

type TunnelCreateUseAllocation struct {
	Type string `json:"type"`    // "dedicated-ip", "port-allocation" or "region"
	Data any    `json:"details"` // UseAllocDedicatedIp, UseAllocPortAlloc, UseRegion
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

func (tun *Tunnel) Create(Token string) error {
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
		ID string `json:"id"`
	}
	if _, err = requestToApi("/tunnels/create", Token, bytes.NewReader(body), &tunnelId, nil); err != nil {
		return err
	}

	id, err := uuid.Parse(tunnelId.ID)
	if err != nil {
		return err
	}
	tun.ID = &id
	return nil
}

func (tun *Tunnel) Delete(Token string) error {
	if tun.ID == nil {
		return nil
	}
	
	body, err := json.Marshal(struct {
		TunnelID uuid.UUID `json:"tunnel_id"`
	}{*tun.ID})
	if err != nil {
		return err
	}

	_, err = requestToApi("/tunnels/delete", Token, bytes.NewReader(body), nil, nil)
	if err == nil {
		// Clean tun id
		tun.ID = nil
	}
	return err
}

type TunnelAllocPort struct{}

type TunnelList struct {
	TCP     TunnelAllocPort `json:"tcp_alloc"`
	UDP     TunnelAllocPort `json:"udp_alloc"`
	Tunnels any             `json:"tunnels"`
}

func ListTunnels(Token string, Agent, Tunnel *uuid.UUID) (*any, error) {
	tunsBody, err := json.Marshal(struct {
		Tunnel *uuid.UUID `json:"tunnel_id"`
		Agent  *uuid.UUID `json:"agent_id"`
	}{Tunnel, Agent})
	if err != nil {
		return nil, err
	}

	var data any
	if _, err = requestToApi("/tunnels/list", Token, bytes.NewReader(tunsBody), &data, nil); err != nil {
		return nil, err
	}

	return &data, nil
}
