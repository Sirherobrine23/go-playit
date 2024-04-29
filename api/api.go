package api

import (
	"net"

	"github.com/google/uuid"
)

const (
	PlayitAPI string = "https://api.playit.gg"

	TunnelTypeMCBedrock string = "minecraft-bedrock" // Minecraft Bedrock server
	TunnelTypeMCJava    string = "minecraft-java"    // Minecraft java server
	TunnelTypeValheim   string = "valheim"           // valheim
	TunnelTypeTerraria  string = "terraria"          // Terraria multiplayer
	TunnelTypeStarbound string = "starbound"         // starbound
	TunnelTypeRust      string = "rust"              // Rust (No programmer language)
	TunnelType7Days     string = "7days"             // 7days
	TunnelTypeUnturned  string = "unturned"          // unturned

	PortTypeBoth string = "both" // Tunnel support tcp and udp protocol
	PortTypeTcp  string = "tcp"  // Tunnel support only tcp protocol
	PortTypeUdp  string = "udp"  // Tunnel support only udp protocol
)

var (
	PortType []string = []string{
		PortTypeBoth,
		PortTypeTcp,
		PortTypeUdp,
	} // Tunnel protocol supports
	TunnelType []string = []string{
		TunnelTypeMCBedrock,
		TunnelTypeMCJava,
		TunnelTypeValheim,
		TunnelTypeTerraria,
		TunnelTypeStarbound,
		TunnelTypeRust,
		TunnelType7Days,
		TunnelTypeUnturned,
	} // Tunnel slice with current supported tunnels
	Regions []string = []string{
		"smart-global",
		"global",
		"north-america",
		"europe",
		"asia",
		"india",
		"south-america",
	}
)

// default
type AssignedDefaultCreate struct {
	LocalIp   net.IPAddr `json:"local_ip"`             // Local ip address
	LocalPort *uint16    `json:"local_port,omitempty"` // Port or nil
}

type AssignedManagedCreate struct {
	AgentID *uuid.UUID `json:"agent_id,omitempty"` // Agent UUID/ID
}

type AssignedAgentCreate struct {
	AgentID uuid.UUID `json:"agent_id"` // Agent UUID/ID
	AssignedDefaultCreate
}

// Agent origin struct
type AgentCreate struct {
	Type string `json:"type"` // Agent type: "default"|"agent"|"managed"
	Data any    `json:"data"`
}
