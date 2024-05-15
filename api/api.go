package api

const (
	PlayitAPI string = "https://api.playit.gg" // Playit API

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

	RegionSmartGlobal  string = "smart-global"
	RegionGlobal       string = "global"
	RegionNorthAmerica string = "north-america"
	RegionEurope       string = "europe"
	RegionAsia         string = "asia"
	RegionIndia        string = "india"
	RegionSouthAmerica string = "south-america"
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
		RegionSmartGlobal,
		RegionGlobal,
		RegionNorthAmerica,
		RegionEurope,
		RegionAsia,
		RegionIndia,
		RegionSouthAmerica,
	} // Regions slice
)
