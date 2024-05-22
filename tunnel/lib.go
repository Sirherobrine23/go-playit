package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type PortProto struct {
	// 1 => "tcp"
	//
	// 2 => "udp"
	//
	// 3 => "both"
	Value string
}

func (w *PortProto) WriteTo(I io.Writer) error {
	switch w.Value {
	case "tcp": return WriteU8(I, 1)
	case "udp": return WriteU8(I, 2)
	case "both": return WriteU8(I, 3)
	}
	return fmt.Errorf("set valid proto")
}
func (w *PortProto) ReadFrom(I io.Reader) error {
	switch ReadU8(I) {
	case 1:
		w.Value = "tcp"
		return nil
	case 2:
		w.Value = "udp"
		return nil
	case 3:
		w.Value = "both"
		return nil
	}
	return fmt.Errorf("invalid proto")
}

type AgentSessionId struct {
	SessionID, AccountID, AgentID uint64
}

func (w *AgentSessionId) WriteTo(I io.Writer) error {
	var err error
	if err = WriteU64(I, w.SessionID); err != nil {
		return err
	} else if err = WriteU64(I, w.AccountID); err != nil {
		return err
	} else if err = WriteU64(I, w.AgentID); err != nil {
		return err
	}

	return nil
}
func (w *AgentSessionId) ReadFrom(I io.Reader) error {
	w.SessionID, w.AccountID, w.AgentID = ReadU64(I), ReadU64(I), ReadU64(I)
	return nil
}

type PortRange struct {
	IP        net.IP
	PortStart uint16
	PortEnd   uint16
	PortProto PortProto
}

func (w *PortRange) WriteTo(I io.Writer) error {
	if err := binary.Write(I, binary.BigEndian, w.IP); err != nil {
		return err
	} else if err := WriteU16(I, w.PortStart); err != nil {
		return err
	} else if err := WriteU16(I, w.PortEnd); err != nil {
		return err
	} else if err := w.PortProto.WriteTo(I); err != nil {
		return err
	}
	return nil
}
func (w *PortRange) ReadFrom(I io.Reader) error {

	return nil
}