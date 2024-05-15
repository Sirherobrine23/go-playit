package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"slices"

	"github.com/google/uuid"
	"sirherobrine23.org/playit-cloud/go-playit/internal/request"
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
	Type string `json:"type"` // "dedicated-ip", "port-allocation" or "region"
	Data any    `json:"data"` // UseAllocDedicatedIp, UseAllocPortAlloc, Region*
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
	TunnelType string                     `json:"tunnel_type,omitempty"` // Tunnel type from TunnelType* const's
	PortType   string                     `json:"port_type"`             // tcp, udp or both
	PortCount  uint16                     `json:"port_count"`
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
	body, err := json.Marshal(&tun)
	if err != nil {
		return err
	}

	res, err := (&request.Request{Base: PlayitAPI, AgentKey: Token, Headers: map[string]string{}}).Request("POST", "/tunnels/create", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	ggo, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(ggo))

	return nil
}

func (tun *Tunnel) Delete(Token string) error {
	if tun.ID == nil {
		return fmt.Errorf("create tunnel to delete or assign tunnel uuid to delete")
	}

	// encode json body
	body, err := json.Marshal(&tun)
	if err != nil {
		return err
	}

	res, err := (&request.Request{Base: PlayitAPI, AgentKey: Token, Headers: map[string]string{}}).Request("POST", "/tunnels/delete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
