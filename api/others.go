package api

import (
	"bytes"
	"encoding/json"
	"net"
	"net/netip"

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
	PortType   PortProto `json:"proto"`       // Port type
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
func (w *Api) AgentInfo() (*AgentRunData, error) {
	var agent AgentRunData
	_, err := w.requestToApi("/agents/rundata", nil, &agent, nil)
	if err != nil {
		return nil, err
	}

	return &agent, nil
}

type AgentRouting struct {
	Agent    uuid.UUID    `json:"agent_id"`
	Targets4 []netip.Addr `json:"targets4"`
	Targets6 []netip.Addr `json:"targets6"`
}

func (w *Api) AgentRoutings(AgentID *uuid.UUID) (*AgentRouting, error) {
	body, err := json.Marshal(struct {
		Agent *uuid.UUID `json:"agent_id,omitempty"`
	}{AgentID})
	if err != nil {
		return nil, err
	}

	var data AgentRouting
	if _, err = w.requestToApi("/agents/routing/get", bytes.NewReader(body), &data, nil); err != nil {
		return nil, err
	}

	return &data, nil
}

type AgentVersion struct {
	Platform string `json:"platform,omitempty"` // linux, freebsd, windows, macos, android, ios, minecraft-plugin, unknown
	Version  string `json:"version"`
}

type PlayitAgentVersion struct {
	Official       bool         `json:"official"`
	DetailsWebsite string       `json:"details_website"`
	Version        AgentVersion `json:"version"`
}

func (w *Api) ProtoRegisterRegister(Client, Tunnel netip.AddrPort) (string, error) {
	type ProtoRegister struct {
		ClientAddr   *netip.AddrPort    `json:"client_addr"`
		TunnelAddr   *netip.AddrPort    `json:"tunnel_addr"`
		AgentVersion PlayitAgentVersion `json:"agent_version"`
	}

	body, err := json.MarshalIndent(ProtoRegister{
		ClientAddr: &Client,
		TunnelAddr: &Tunnel,
		AgentVersion: PlayitAgentVersion{
			Official:       false,
			DetailsWebsite: "https://sirherobrine23.org/playit-cloud/go-playit",
			Version: AgentVersion{
				Version:  GoPlayitVersion,
				Platform: "unknown",
			},
		},
	}, "", "  ")
	if err != nil {
		return "", err
	}
	var code struct {
		Key string `json:"key"`
	}
	if _, err = w.requestToApi("/proto/register", bytes.NewBuffer(body), &code, nil); err != nil {
		return "", err
	}
	return code.Key, nil
}
