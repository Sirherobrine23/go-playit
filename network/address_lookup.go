package network

import (
	"encoding/binary"
	"net/netip"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type AddressValue[V any] struct {
	Value            V
	FromPort, ToPort uint16
}

type AddressLookup[Value any] interface {
	Lookup(IP netip.Addr, Port uint16, Proto api.PortProto) *AddressValue[Value]
}

type MatchIP struct {
	IPNumber uint64
	RegionID *uint16
}

func (slef *MatchIP) Matches(ip netip.Addr) bool {
	if ip.Is6() {
		other := NewMatchIP(ip)
		return slef.IPNumber == other.IPNumber && slef.RegionID == other.RegionID
	}
	octs := ip.As4()
	if uint64(octs[3]) != slef.IPNumber {
		return false
	}
	if slef.RegionID == nil {
		return true
	}
	return RegionNumberV4(ip) == *slef.RegionID
}

func NewMatchIP(ip netip.Addr) MatchIP {
	parts := ip.As16()
	regionID := binary.BigEndian.Uint16([]byte{parts[6], parts[7]})
	ipNumber := binary.BigEndian.Uint64([]byte{
		parts[8],
		parts[9],
		parts[10],
		parts[11],
		parts[12],
		parts[13],
		parts[14],
		parts[15],
	})
	info := MatchIP{IPNumber: ipNumber}
	if regionID != 0 {
		info.RegionID = new(uint16)
		*info.RegionID = regionID
	}
	return info
}

func RegionNumberV4(ip netip.Addr) uint16 {
	octs := ip.As4();
	if octs[0] == 147 && octs[1] == 185 && octs[2] == 221 { /* 147.185.221.0/24 (1) */
		return 1
	} else if octs[0] == 209 && octs[1] == 25 && octs[2] >= 140 && octs[2] <= 143 { /* 209.25.140.0/22 (2 to 5) */
		return uint16(2 + (octs[2] - 140))
	} else if octs[0] == 23 && octs[1] == 133 && octs[2] == 216 { /* 23.133.216.0/24 (6) */
		return 6
	}
	/* global IP */
	return 0
}

type MappingOverride struct {
	Proto     api.PortProto
	Port      api.PortRange
	LocalAddr netip.AddrPort
	MatchIP   MatchIP
}

type LookupWithOverrides []MappingOverride

func (look *LookupWithOverrides) Lookup(IP netip.Addr, Port uint16, Proto api.PortProto) *AddressValue[netip.AddrPort] {
	for _, over := range *look {
		if (over.Port.From <= Port && Port < over.Port.To) && (over.Proto == "both" || over.Proto == Proto) {
			return &AddressValue[netip.AddrPort]{
				Value: over.LocalAddr,
				FromPort: over.Port.From,
				ToPort: over.Port.To,
			}
		}
	}
	return &AddressValue[netip.AddrPort]{
		Value: netip.AddrPortFrom(netip.IPv4Unspecified(), Port),
		FromPort: Port,
		ToPort: Port+1,
	}
}