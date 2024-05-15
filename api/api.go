package api

const (
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

	RegionGlobal       string = "global"        // Free account and premium
	RegionSmartGlobal  string = "smart-global"  // Require premium account
	RegionNorthAmerica string = "north-america" // Require premium account
	RegionEurope       string = "europe"        // Require premium account
	RegionAsia         string = "asia"          // Require premium account
	RegionIndia        string = "india"         // Require premium account
	RegionSouthAmerica string = "south-america" // Require premium account
)

var (
	PlayitAPI string   = "https://api.playit.gg" // Playit API
	PortType  []string = []string{
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
		RegionSmartGlobal,
		RegionGlobal,
		RegionNorthAmerica,
		RegionEurope,
		RegionAsia,
		RegionIndia,
		RegionSouthAmerica,
	} // Regions slice
)
