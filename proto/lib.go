package proto

import (
	"fmt"
	"io"
	"net/netip"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
)

type AgentSessionId struct {
	SessionID, AccountID, AgentID uint64
}

type PortRange struct {
	IP                 netip.Addr
	PortStart, PortEnd uint16
	PortProto          PortProto
}

type PortProto string

func (AgentSession *AgentSessionId) WriteTo(w io.Writer) error {
	if err := enc.WriteU64(w, AgentSession.SessionID); err != nil {
		return err
	} else if err := enc.WriteU64(w, AgentSession.AccountID); err != nil {
		return err
	} else if err := enc.WriteU64(w, AgentSession.AgentID); err != nil {
		return err
	}
	return nil
}
func (AgentSession *AgentSessionId) ReadFrom(r io.Reader) error {
	AgentSession.SessionID, AgentSession.AccountID, AgentSession.AgentID = enc.ReadU64(r), enc.ReadU64(r), enc.ReadU64(r)
	return nil
}

func (portRange *PortRange) WriteTo(w io.Writer) error {
	if err := enc.AddrWrite(w, portRange.IP); err != nil {
		return err
	} else if err := enc.WriteU16(w, portRange.PortStart); err != nil {
		return err
	} else if err := enc.WriteU16(w, portRange.PortEnd); err != nil {
		return err
	} else if err := portRange.PortProto.WriteTo(w); err != nil {
		return err
	}
	return nil
}
func (portRange *PortRange) ReadFrom(r io.Reader) error {
	var err error
	portRange.IP, err = enc.AddrRead(r)
	if err != nil {
		return err
	}
	portRange.PortStart, portRange.PortEnd = enc.ReadU16(r), enc.ReadU16(r)
	portRange.PortProto = PortProto("")
	if err := portRange.PortProto.ReadFrom(r); err != nil {
		return err
	}
	return nil
}

func (proto PortProto) WriteTo(w io.Writer) error {
	switch proto {
	case "tcp":
		return enc.WriteU8(w, 1)
	case "udp":
		return enc.WriteU8(w, 2)
	case "both":
		return enc.WriteU8(w, 3)
	}
	return fmt.Errorf("invalid port proto")
}
func (proto PortProto) ReadFrom(r io.Reader) error {
	switch enc.ReadU8(r) {
	case 1:
		proto = PortProto("tcp")
	case 2:
		proto = PortProto("udp")
	case 3:
		proto = PortProto("both")
	default:
		return fmt.Errorf("invalid port proto")
	}
	return nil
}
