package network

import (
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