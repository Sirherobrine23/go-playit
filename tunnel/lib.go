package tunnel

import (
	"fmt"
	"io"
	"net/netip"
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

func (AgentSession *AgentSessionId) WriteTo(w io.Writer) (int64, error) {
	if _, err := writeU64(w, AgentSession.SessionID); err != nil {
		return 0, err
	} else if _, err = writeU64(w, AgentSession.AccountID); err != nil {
		return 8, err
	} else if _, err = writeU64(w, AgentSession.AgentID); err != nil {
		return 16, err
	}
	return 24, nil
}
func (AgentSession *AgentSessionId) ReadFrom(r io.Reader) (int64, error) {
	AgentSession.SessionID, AgentSession.AccountID, AgentSession.AgentID = readU64(r), readU64(r), readU64(r)
	return 24, nil
}

func (portRange *PortRange) WriteTo(w io.Writer) (int64, error) {
	var len int64 = 4
	sizeIP, err := addrWrite(w, portRange.IP)
	if err != nil {
		return len, err
	}
	len += sizeIP
	if _, err = writeU16(w, portRange.PortStart); err != nil {
		return len, err
	} else if _, err = writeU16(w, portRange.PortEnd); err != nil {
		return len, err
	}
	protoSize, err := portRange.PortProto.WriteTo(w)
	if err != nil {
		return len, err
	}
	return len + protoSize, nil
}
func (portRange *PortRange) ReadFrom(r io.Reader) (int64, error) {
	var (
		ipSize int64
		err    error
	)
	portRange.IP, ipSize, err = addrRead(r)
	if err != nil {
		return ipSize, err
	}

	ipSize += 4
	portRange.PortStart, portRange.PortEnd = readU16(r), readU16(r)

	portRange.PortProto = PortProto("")
	protoSize, err := portRange.PortProto.ReadFrom(r)
	if err != nil {
		return ipSize, err
	}
	return ipSize + protoSize, nil
}

func (proto PortProto) WriteTo(w io.Writer) (int64, error) {
	switch proto {
	case "tcp":
		return writeU8(w, 1)
	case "udp":
		return writeU8(w, 2)
	case "both":
		return writeU8(w, 3)
	}
	return 0, fmt.Errorf("invalid port proto")
}
func (proto PortProto) ReadFrom(r io.Reader) (int64, error) {
	switch readU8(r) {
	case 1:
		proto = PortProto("tcp")
	case 2:
		proto = PortProto("udp")
	case 3:
		proto = PortProto("both")
	default: return 0, fmt.Errorf("invalid port proto")
	}
	return 1, nil
}
