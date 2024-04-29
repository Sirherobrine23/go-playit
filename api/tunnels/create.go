package tunnels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"sirherobrine23.org/playit-cloud/go-agent/api"
	"sirherobrine23.org/playit-cloud/go-agent/internal/request"
)

type UseAllocDedicatedIp struct {
	IPaddress string  `json:"ip_hostname"`
	Port      *uint16 `json:"port,omitempty"`
}

type UseAllocPortAlloc struct {
	ID uuid.UUID `json:"alloc_id"`
}

type UseRegion struct {
	// "smart-global"|"global"|"north-america"|"europe"|"asia"|"india"|"south-america"
	Region string `json:"region"`
}

type AgentAllocCreate struct {
	Type    string `json:"type"`    // Allocation type: "dedicated-ip"|"port-allocation"|"region"
	Details any    `json:"details"` // allocation details
}

type CreateTunnel struct {
	TunnelType *string           `json:"tunnel_type,omitempty"` // Tunnel type create
	FirewallID *string           `json:"firewall_id,omitempty"` // Firewall id
	Enabled    bool              `json:"enabled"`               // Tunnel is enabled
	PortType   string            `json:"port_type"`             // port type to watch
	Port       int               `json:"port_count"`            // Port
	Origin     api.AgentCreate   `json:"origin"`                // Tunnel origin
	Alloc      *AgentAllocCreate `json:"alloc,omitempty"`       // Alloc type
}

// Create tunnel
func Create(secret string, tun CreateTunnel) (*string, error) {
	if tun.TunnelType != nil && !slices.Contains(api.TunnelType, *tun.TunnelType) {
		// Return error if tunnel type not contains in slice type
		return nil, fmt.Errorf("invalid tunnel type")
	} else if slices.Contains(api.PortType, tun.PortType) {
		// return error if invalid port type is informed
		return nil, fmt.Errorf("invalid port type")
	}

	// Set agent origin
	if _, ok := tun.Origin.Data.(api.AssignedDefaultCreate); ok {
		tun.Origin.Type = "default"
	} else if _, ok := tun.Origin.Data.(api.AssignedManagedCreate); ok {
		tun.Origin.Type = "managed"
	} else if _, ok := tun.Origin.Data.(api.AssignedAgentCreate); ok {
		tun.Origin.Type = "agent"
	} else {
		// Return error for invalid Agent type
		return nil, fmt.Errorf("invalid agent")
	}

	if tun.Alloc != nil {
		switch data := tun.Alloc.Details.(type) {
		case UseRegion:
			tun.Alloc.Type = "region"
			if !slices.Contains(api.Regions, data.Region) {
				data.Region = "smart-global"
				tun.Alloc.Details = data
			}
		case UseAllocDedicatedIp:
			tun.Alloc.Type = "dedicated-ip"
		case UseAllocPortAlloc:
			tun.Alloc.Type = "port-allocation"
		default:
			tun.Alloc = nil
		}
	}

	body, err := json.MarshalIndent(&tun, "", "  ")
	if err != nil {
		return nil, err
	}

	req := request.RequestOptions{
		Method: "POST",
		Url:    fmt.Sprintf("%s/tunnels/create", api.PlayitAPI),
		Body:   bytes.NewReader(body),
		Headers: http.Header{
			"x-content-type": {"application/json"},
			"x-accepts":      {"application/json"},
		},
	}
	if secret = strings.TrimSpace(secret); len(secret) > 0 {
		req.Headers.Set("Authorization", fmt.Sprintf("Agent-Key %s", secret))
	}

	var status struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
	}
	res, err := req.Do(&status)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		info := status.Data.(struct{ id string })
		return &info.id, nil
	} else if res.StatusCode == 400 {
		info := status.Data.(struct{ message string })
		return nil, fmt.Errorf(info.message)
	} else if res.StatusCode == 401 {
		return nil, fmt.Errorf("invaid secret")
	}
	return nil, fmt.Errorf("backend error, code: %d (%s)", res.StatusCode, res.Status)
}
