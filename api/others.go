package api

import (
	"bytes"
	"encoding/json"
	"net"

	"github.com/google/uuid"
)

type PortRange struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}

type AgentTunnel struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	IpNum          uint16    `json:"ip_num"`
	RegionNum      uint16    `json:"region_num"`
	Port           PortRange `json:"port"`
	Proto          string    `json:"proto"`
	LocalIp        net.IP    `json:"local_ip"`
	LocalPort      uint16    `json:"local_port"`
	TunnelType     string    `json:"tunnel_type"`
	AssignedDomain string    `json:"assigned_domain"`
	CustomDomain   string    `json:"custom_domain"`
	Disabled       *any      `json:"disabled"`
}

type AgentPendingTunnel struct {
	ID         uuid.UUID `json:"id"`          // Agent ID
	Name       string    `json:"name"`        // Agent Name
	PortType   string    `json:"proto"`       // Port type
	PortCount  uint16    `json:"port_count"`  // Port count
	TunnelType string    `json:"tunnel_type"` // Tunnel type
	Disabled   bool      `json:"is_disabled"` // Tunnel is disabled
}

type AgentRunData struct {
	ID             uuid.UUID            `json:"agent_id"`
	Type           string               `json:"agent_type"`
	AccountStatus  string               `json:"account_status"` // "account-delete-scheduled", "banned", "has-message", "email-not-verified", "guest", "ready", "agent-over-limit" or "agent-disabled"
	Tunnels        []AgentTunnel        `json:"tunnels"`
	TunnelsPending []AgentPendingTunnel `json:"pending"`
}

// Get agent info
func AgentInfo(Token string) (*AgentRunData, error) {
	var agent AgentRunData
	_, err := requestToApi("/agents/rundata", Token, nil, &agent, nil)
	if err != nil {
		return nil, err
	}

	return &agent, nil
}

type AgentRouting struct {
	Agent    uuid.UUID `json:"agent_id"`
	Targets4 []net.IP  `json:"targets4"`
	Targets6 []net.IP  `json:"targets6"`
}

func AgentRoutings(Token string, AgentID *uuid.UUID) (*AgentRouting, error) {
	body, err := json.Marshal(struct {
		Agent *uuid.UUID `json:"agent_id,omitempty"`
	}{AgentID})
	if err != nil {
		return nil, err
	}

	var data AgentRouting
	if _, err = requestToApi("/agents/routing/get", Token, bytes.NewReader(body), &data, nil); err != nil {
		return nil, err
	}

	return &data, nil
}
