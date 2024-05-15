package api

import (
	"encoding/json"
	"net"

	"github.com/google/uuid"
	"sirherobrine23.org/playit-cloud/go-playit/internal/request"
)

type PortRange struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}

type AgentTunnel struct {
	ID             uuid.UUID  `json:"id"`
	Name           *string    `json:"name"`
	IpNum          uint16     `json:"ip_num"`
	RegionNum      uint16     `json:"region_num"`
	Port           PortRange  `json:"port"`
	Proto          string     `json:"proto"`
	LocalIp        net.IPAddr `json:"local_ip"`
	LocalPort      uint16     `json:"local_port"`
	TunnelType     string     `json:"tunnel_type"`
	AssignedDomain string     `json:"assigned_domain"`
	CustomDomain   *string    `json:"custom_domain"`
	Disabled       *any       `json:"disabled"`
}

type AgentPendingTunnel struct {
	ID         uuid.UUID `json:"id"`
	Name       *string   `json:"name"`
	PortType   string    `json:"proto"`
	PortCount  uint16    `json:"port_count"`
	TunnelType *string   `json:"tunnel_type"`
	Disabled   bool      `json:"is_disabled"`
}

type AgentRunData struct {
	ID             uuid.UUID            `json:"agent_id"`
	Type           string               `json:"agent_type"`
	AccountStatus  string               `json:"account_status"` // "account-delete-scheduled", "banned", "has-message", "email-not-verified", "guest", "ready", "agent-over-limit" or "agent-disabled"
	Tunnels        []AgentTunnel        `json:"tunnels"`
	TunnelsPending []AgentPendingTunnel `json:"pending"`
}

func AgentInfo(Token string) (*AgentRunData, error) {
	res, err := (&request.Request{Base: PlayitAPI, Token: Token, Headers: map[string]string{}}).Request("POST", "/agents/rundata", nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var agent AgentRunData
	if err = json.NewDecoder(res.Body).Decode(&agent); err != nil {
		return nil, err
	}

	return &agent, nil
}
