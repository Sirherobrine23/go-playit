package tunnels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"sirherobrine23.org/playit-cloud/go-agent/api"
	"sirherobrine23.org/playit-cloud/go-agent/internal/request"
)

type AllocatedPort struct {
	Allowed uint16 `json:"allowed"`
	Claimed uint16 `json:"claimed"`
	Desired uint16 `json:"desired"`
}

type TunnelAllocated struct {
	ID         string   `json:"id"`
	IpHostname string   `json:"ip_hostname"`
	StaticIPV4 string   `json:"static_ip4"`
	Domain     string   `json:"assigned_domain"`
	SRV        string   `json:"assigned_srv"`
	Region     string   `json:"region"`
	IP         net.Addr `json:"tunnel_ip"`
	IpType     string   `json:"ip_type"` // "both" | "ip4" | "ip6"
	PortStart  int      `json:"port_start"`
	PortEnd    int      `json:"port_end"`
}

type AccountTunnel struct {
	ID            uuid.UUID       `json:"id"`
	Active        bool            `json:"active"`
	FirewallID    *uuid.UUID      `json:"firewall_id,omitempty"`
	Name          string          `json:"name"`
	Region        string          `json:"region"`
	TunnelType    string          `json:"tunnel_type"`
	CreateTime    time.Time       `json:"created_at"`
	PortType      string          `json:"port_type"`
	PortCount     int             `json:"port_count"`
	DisableReason string          `json:"disabled_reason"` // "requires-premium" | "over-port-limit" | "ip-used-in-gre"
	Origin        api.AgentCreate `json:"origin"`
	Expire        struct {
		Disable time.Time `json:"disable_at"`
		Remove  time.Time `json:"remove_at"`
	} `json:"expire_notice"`
	Ratelimit struct {
		BytesPerSec   int64 `json:"bytes_per_second"`
		PacketsPerSec int64 `json:"packets_per_second"`
	} `json:"ratelimit"`
	Domain *struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Parent   *string `json:"parent"`
		Source   string  `json:"source"` // "from-ip" | "from-tunnel" | "from-agent-ip"
		External bool    `json:"is_external"`
	} `json:"domain"`
	Alloc struct {
		Status string `json:"status"` // "disabled" | "allocated"
		Data   any    `json:"data"`
	} `json:"alloc"`
}

type AccountTunnels struct {
	TCP     []AllocatedPort `json:"tcp_alloc"`
	UDP     []AllocatedPort `json:"udp_alloc"`
	Tunnels []AccountTunnel `json:"tunnels"`
}

func List(secret string, tunID string) (*AccountTunnels, error) {
	body, err := json.MarshalIndent(struct {
		Tunnel string `json:"tunnel_id"`
	}{tunID}, "", "  ")
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
		info := status.Data.(AccountTunnels)
		return &info, nil
	} else if res.StatusCode == 400 {
		info := status.Data.(struct{ message string })
		return nil, fmt.Errorf(info.message)
	} else if res.StatusCode == 401 {
		return nil, fmt.Errorf("invaid secret")
	}
	return nil, fmt.Errorf("backend error, code: %d (%s)", res.StatusCode, res.Status)
}
