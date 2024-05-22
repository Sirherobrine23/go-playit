package tunnel

import (
	"net/netip"
	"slices"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type PortType struct {
	Value string
}
func (w *PortType) IsValid() bool {
	return slices.Contains(api.PortType, w.Value)
}
func (proto *PortProto) SetBoth() {
	proto.Value = "both"
}
func (proto *PortProto) SetTcp() {
	proto.Value = "tcp"
}
func (proto *PortProto) SetUdp() {
	proto.Value = "udp"
}

type AddressValue[T any] struct {
	Value            T
	FromPort, ToPort uint16
}

type AddressLookup[T any] interface {
	// Resolve address if exist return value else return nil point
	Lookup(IpPort netip.AddrPort, Proto PortType) *AddressValue[T]
}

type MatchIp struct {
	IP netip.AddrPort
	RegionID *uint16
}
func (mat *MatchIp) Matches(ip netip.AddrPort) bool {
	return mat.IP.Compare(ip) == 0
}

type MappingOverride struct {
	MatchIP   MatchIp
	Proto     PortType
	Port      api.PortRange
	LocalAddr netip.AddrPort
}

type LookupWithOverrides []MappingOverride

func (Look *LookupWithOverrides) Lookup(IpPort netip.AddrPort, Proto PortType) *AddressValue[netip.AddrPort] {
	for _, Over := range *Look {
		if Over.Proto.Value == Proto.Value && Over.MatchIP.Matches(IpPort) {
			return &AddressValue[netip.AddrPort]{
				Value: Over.LocalAddr,
				FromPort: Over.Port.From,
				ToPort: Over.Port.To,
			}
		}
	}
	return &AddressValue[netip.AddrPort]{
		Value:    netip.AddrPortFrom(netip.AddrFrom4([4]byte{127, 0, 0, 1}), IpPort.Port()),
		FromPort: IpPort.Port(),
		ToPort:   IpPort.Port() + 1,
	}
}
