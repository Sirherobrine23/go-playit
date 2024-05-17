package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

func readU8(I io.Reader) uint8 {
	var value uint8
	binary.Read(I, binary.BigEndian, &value)
	return value
}

func readU16(I io.Reader) uint16 {
	var value uint16
	binary.Read(I, binary.BigEndian, &value)
	return value
}

func readU32(I io.Reader) uint32 {
	var value uint32
	binary.Read(I, binary.BigEndian, &value)
	return value
}

func readU64(I io.Reader) uint64 {
	var value uint64
	binary.Read(I, binary.BigEndian, &value)
	return value
}

func ReadOption(I io.Reader, value func(I io.Reader) error) error {
	switch readU8(I) {
	case 0:
		return nil
	case 1:
		return value(I)
	}
	return nil
}

type Socket struct {
	Ip   net.IP
	Port uint16
}

func ReadAddress(I io.Reader) (*Socket, error) {
	sock := Socket{}
	switch readU8(I) {
	case 4:
		sock.Ip = net.IPv4(readU8(I), readU8(I), readU8(I), readU8(I))
		sock.Port = readU16(I)
		return &sock, nil
	case 6:
		buff := make([]uint8, 16)
		if _, err := I.Read(buff); err != nil {
			return nil, err
		}
		sock.Ip = sock.Ip
		sock.Port = readU16(I)
		return &sock, nil
	}
	return nil, fmt.Errorf("cannot get IP type")
}

type MessageEncoding interface {
	WriteTo(I io.Writer) error
	ReadFrom(I io.Reader) error
}

type ControlRpcMessage struct {
	RequestID uint64
	Content   any
}

func (Control *ControlRpcMessage) WriteTo(I io.Writer) error {
	if err := binary.Write(I, binary.BigEndian, Control.RequestID); err != nil {
		return err
	}
	return Control.Content.(MessageEncoding).WriteTo(I)
}

func (Control *ControlRpcMessage) ReadFrom(I io.Reader) error {
	Control.RequestID = readU64(I)
	if Control.Content == nil {
		return nil
	}
	return Control.Content.(MessageEncoding).ReadFrom(I)
}

type ControlRequest struct {
	MessageEncoding
	Data any
}

func (w *ControlRequest) WriteTo(I io.Writer) error {
	if pingInfo, isPing := w.Data.(Ping); isPing {
		if err := binary.Write(I, binary.BigEndian, uint32(6)); err != nil {
			return err
		}
		return pingInfo.WriteTo(I)
	}

	return nil
}

func (w *ControlRequest) ReadFrom(I io.Reader) error {
	var d uint32
	err := binary.Read(I, binary.BigEndian, &d)
	if err != nil {
		return err
	}

	switch d {
	case 1:
		data := Ping{}
		err = data.ReadFrom(I)
	case 2:
	case 3:
	case 4:
	case 5:
		return nil
	default:
		err = fmt.Errorf("invalid ControlRequest id")
	}

	return err
}

type AgentSessionId struct {
	MessageEncoding
	SessionID uint64
	AccountID uint64
	AgentID   uint64
}

func (w *AgentSessionId) WriteTo(I io.Writer) error {
	if err := binary.Write(I, binary.BigEndian, &w.SessionID); err != nil {
		return err
	} else if err := binary.Write(I, binary.BigEndian, &w.AccountID); err != nil {
		return err
	} else if err := binary.Write(I, binary.BigEndian, &w.AgentID); err != nil {
		return err
	}
	return nil
}

func (w *AgentSessionId) ReadFrom(I io.Reader) error {
	if err := binary.Read(I, binary.BigEndian, &w.SessionID); err != nil {
		return err
	} else if err := binary.Read(I, binary.BigEndian, &w.AccountID); err != nil {
		return err
	} else if err := binary.Read(I, binary.BigEndian, &w.AgentID); err != nil {
		return err
	}
	return nil
}

type Ping struct {
	Now         time.Time
	CurrentPing *uint32
	SessionID   *AgentSessionId
}

func (ping *Ping) WriteTo(I io.Writer) error {
	if err := binary.Write(I, binary.BigEndian, uint64(ping.Now.UnixMilli())); err != nil {
		return err
	}

	if ping.CurrentPing == nil {
		if err := binary.Write(I, binary.BigEndian, uint8(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(I, binary.BigEndian, uint8(1)); err != nil {
			return err
		} else if err := binary.Write(I, binary.BigEndian, ping.CurrentPing); err != nil {
			return err
		}
	}

	if ping.SessionID == nil {
		if err := binary.Write(I, binary.BigEndian, uint8(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(I, binary.BigEndian, uint8(1)); err != nil {
			return err
		} else if err := ping.SessionID.WriteTo(I); err != nil {
			return err
		}
	}

	return nil
}

func (ping *Ping) ReadFrom(I io.Reader) error {
	if err := binary.Read(I, binary.BigEndian, &ping.Now); err != nil {
		return err
	} else if err := binary.Read(I, binary.BigEndian, &ping.CurrentPing); err != nil {
		return err
	} else if err := ping.SessionID.ReadFrom(I); err != nil {
		return err
	}
	return nil
}

type Pong struct {
	MessageEncoding
	RequestNow, ServerNow uint64
	ServerID              uint64
	DataCenterId          uint32
	ClientAddress         *Socket
	TunnelAddress         *Socket
	SessionExpire         uint64
}

func (w *Pong) ReadFrom(I io.Reader) error {
	var err error
	w.RequestNow = readU64(I)
	w.ServerNow = readU64(I)
	w.ServerID = readU64(I)
	w.DataCenterId = readU32(I)
	w.ClientAddress, err = ReadAddress(I)
	if err != nil {
		return err
	}
	w.TunnelAddress, err = ReadAddress(I)
	if err != nil {
		return err
	}
	return ReadOption(I, func(I io.Reader) error {
		w.SessionExpire = readU64(I)
		return nil
	})
}

type ControlFeed struct {
	MessageEncoding
	Data struct{ ControlRpcMessage }
}

func (w *ControlFeed) ReadFrom(I io.Reader) error {
	switch readU32(I) {
	case 1:
		w.Data = struct{ ControlRpcMessage }{ControlRpcMessage{
			Content: &ControlResponse{},
		}}
		return w.Data.ControlRpcMessage.ReadFrom(I)
	case 2:
		return fmt.Errorf("client not implemented")
	default:
		return fmt.Errorf("invalid controlFeed id")
	}
}

type ControlResponse struct {
	Data any
}

func (w *ControlResponse) WriteTo(I io.Writer) error {
	return nil
}

func (w *ControlResponse) ReadFrom(I io.Reader) error {
	switch readU32(I) {
	case 1:
		data := Pong{}
		err := data.ReadFrom(I)
		w.Data = data
		return err
	}
	return fmt.Errorf("Invalid ControlResponse")
}
