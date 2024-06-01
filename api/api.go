package api

import (
	"log"
	"os"
	"runtime"
)

var NullFile, _ = os.Open(os.DevNull)
var debug = log.New(NullFile, "api.playit.gg: ", log.Ldate)

const (
	GoPlayitVersion string = "0.17.1"
	PlayitAPI       string = "https://api.playit.gg" // Playit API

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

type Api struct {
	Code   string // Claim code
	Secret string // Agent Secret
}

type PortProto string

func (proto PortProto) IsValid() bool {
	switch proto {
	case PortProto(PortTypeBoth):
	case PortProto(PortTypeTcp):
	case PortProto(PortTypeUdp):
		return true
	}
	return false
}
func (proto PortProto) SetBoth() {
	proto = "both"
}
func (proto PortProto) SetTcp() {
	proto = "tcp"
}
func (proto PortProto) SetUdp() {
	proto = "udp"
}

type Platform string

func (Platform Platform) Host() {
	switch runtime.GOOS {
	case "linux":
		Platform = "linux"
	case "freebsd":
		Platform = "freebsd"
	case "windows":
		Platform = "windows"
	case "darwin":
		Platform = "macos"
	case "android":
		Platform = "android"
	case "ios":
		Platform = "ios"
	default:
		Platform = "unknown"
	}
}
func (Platform Platform) Linux() {
	Platform = "linux"
}
func (Platform Platform) Freebsd() {
	Platform = "freebsd"
}
func (Platform Platform) Windows() {
	Platform = "windows"
}
func (Platform Platform) Macos() {
	Platform = "macos"
}
func (Platform Platform) Android() {
	Platform = "android"
}
func (Platform Platform) Ios() {
	Platform = "ios"
}
func (Platform Platform) MinecraftPlugin() {
	Platform = "minecraft-plugin"
}
func (Platform Platform) Unknown() {
	Platform = "unknown"
}
