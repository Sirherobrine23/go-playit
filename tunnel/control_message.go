package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type Ping struct {
	Now         time.Time
	CurrentPing *uint32
	SessionID   *AgentSessionId
}

func (w *Ping) WriteTo(I io.Writer) error {
	if err := WriteU64(I, uint64(w.Now.UnixMilli())); err != nil {
		return err
	}

	if w.CurrentPing == nil {
		if err := binary.Write(I, binary.BigEndian, uint8(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(I, binary.BigEndian, uint8(1)); err != nil {
			return err
		} else if err := binary.Write(I, binary.BigEndian, w.CurrentPing); err != nil {
			return err
		}
	}

	if w.SessionID == nil {
		return binary.Write(I, binary.BigEndian, uint8(0))
	} else if err := binary.Write(I, binary.BigEndian, uint8(1)); err != nil {
		return err
	}
	return w.SessionID.WriteTo(I)
}
func (w *Ping) ReadFrom(I io.Reader) error {
	w.Now = time.UnixMilli(int64(ReadU64(I)))

	CurrentPing := ReadU32(I)
	w.CurrentPing = &CurrentPing

	w.SessionID = &AgentSessionId{}
	w.SessionID.ReadFrom(I)
	return nil
}

type Pong struct {
	RequestNow      uint64
	ServerNow       uint64
	ServerId        uint64
	DataCenterId    uint32
	ClientAddr      AddressPort
	TunnelAddr      AddressPort
	SessionExpireAt *uint64
}

func (w *Pong) WriteTo(I io.Writer) error {
	if err := WriteU64(I, w.RequestNow); err != nil {
		return err
	} else if err := WriteU64(I, w.ServerNow); err != nil {
		return err
	} else if err := WriteU64(I, w.ServerId); err != nil {
		return err
	} else if err := WriteU32(I, w.DataCenterId); err != nil {
		return err
	} else if err := w.ClientAddr.WriteTo(I); err != nil {
		return err
	} else if err := w.TunnelAddr.WriteTo(I); err != nil {
		return err
	} else if err := WriteOptionU64(I, w.SessionExpireAt); err != nil {
		return err
	}
	return nil
}
func (w *Pong) ReadFrom(I io.Reader) error {
	w.RequestNow, w.ServerNow, w.ServerId = ReadU64(I), ReadU64(I), ReadU64(I)
	w.DataCenterId = ReadU32(I)

	w.ClientAddr = AddressPort{}
	w.ClientAddr.ReadFrom(I)

	w.TunnelAddr = AddressPort{}
	w.TunnelAddr.ReadFrom(I)

	Sess := ReadU64(I)
	w.SessionExpireAt = &Sess
	return nil
}

type AgentRegister struct{}

func (w *AgentRegister) WriteTo(I io.Writer) error  { return nil }
func (w *AgentRegister) ReadFrom(I io.Reader) error { return nil }

type AgentCheckPortMapping struct{}

func (w *AgentCheckPortMapping) WriteTo(I io.Writer) error  { return nil }
func (w *AgentCheckPortMapping) ReadFrom(I io.Reader) error { return nil }

type ControlRequest struct {
	Ping                  *Ping
	AgentRegister         *AgentRegister
	AgentKeepAlive        *AgentSessionId
	SetupUdpChannel       *AgentSessionId
	AgentCheckPortMapping *AgentCheckPortMapping
}

func (w *ControlRequest) WriteTo(I io.Writer) error {
	if w.Ping != nil {
		if err := WriteU32(I, uint32(6)); err != nil {
			return err
		}
		return w.Ping.WriteTo(I)
	} else if w.AgentRegister != nil {
		if err := WriteU32(I, uint32(2)); err != nil {
			return err
		}
		return w.AgentRegister.WriteTo(I)
	} else if w.AgentKeepAlive != nil {
		if err := WriteU32(I, uint32(3)); err != nil {
			return err
		}
		return w.AgentKeepAlive.WriteTo(I)
	} else if w.SetupUdpChannel != nil {
		if err := WriteU32(I, uint32(4)); err != nil {
			return err
		}
		return w.SetupUdpChannel.WriteTo(I)
	} else if w.AgentCheckPortMapping != nil {
		if err := WriteU32(I, uint32(5)); err != nil {
			return err
		}
		return w.AgentCheckPortMapping.WriteTo(I)
	}
	return fmt.Errorf("set ControlRequest")
}
func (w *ControlRequest) ReadFrom(I io.Reader) error {
	switch ReadU32(I) {
	case 1:
		w.Ping = &Ping{}
		return w.Ping.ReadFrom(I)
	case 2:
		w.AgentRegister = &AgentRegister{}
		return w.AgentRegister.ReadFrom(I)
	case 3:
		w.AgentKeepAlive = &AgentSessionId{}
		return w.AgentKeepAlive.ReadFrom(I)
	case 4:
		w.SetupUdpChannel = &AgentSessionId{}
		return w.SetupUdpChannel.ReadFrom(I)
	case 5:
		w.AgentCheckPortMapping = &AgentCheckPortMapping{}
		return w.AgentCheckPortMapping.ReadFrom(I)
	}
	return fmt.Errorf("invalid ControlRequest id")
}

type AgentRegistered struct{}

func (w *AgentRegistered) WriteTo(I io.Writer) error  { return nil }
func (w *AgentRegistered) ReadFrom(I io.Reader) error { return nil }

type AgentPortMapping struct{}

func (w *AgentPortMapping) WriteTo(I io.Writer) error  { return nil }
func (w *AgentPortMapping) ReadFrom(I io.Reader) error { return nil }

type UdpChannelDetails struct{}

func (w *UdpChannelDetails) WriteTo(I io.Writer) error  { return nil }
func (w *UdpChannelDetails) ReadFrom(I io.Reader) error { return nil }

type ControlResponse struct {
	Pong              *Pong
	InvalidSignature  bool
	Unauthorized      bool
	RequestQueued     bool
	TryAgainLater     bool
	AgentRegistered   *AgentRegistered
	AgentPortMapping  *AgentPortMapping
	UdpChannelDetails *UdpChannelDetails
}

func (w *ControlResponse) WriteTo(I io.Writer) error {
	if w.Pong != nil {
		if err := WriteU32(I, 1); err != nil {
			return err
		}
		return w.Pong.WriteTo(I)
	} else if w.InvalidSignature {
		if err := WriteU32(I, 2); err != nil {
			return err
		}
		return nil
	} else if w.Unauthorized {
		if err := WriteU32(I, 3); err != nil {
			return err
		}
		return nil
	} else if w.RequestQueued {
		if err := WriteU32(I, 4); err != nil {
			return err
		}
		return nil
	} else if w.TryAgainLater {
		if err := WriteU32(I, 5); err != nil {
			return err
		}
		return nil
	} else if w.AgentRegistered != nil {
		if err := WriteU32(I, 6); err != nil {
			return err
		}
		return w.AgentRegistered.WriteTo(I)
	} else if w.AgentPortMapping != nil {
		if err := WriteU32(I, 7); err != nil {
			return err
		}
		return w.AgentPortMapping.WriteTo(I)
	} else if w.UdpChannelDetails != nil {
		if err := WriteU32(I, 8); err != nil {
			return err
		}
		return w.UdpChannelDetails.WriteTo(I)
	}
	return nil
}
func (w *ControlResponse) ReadFrom(I io.Reader) error {
	switch ReadU32(I) {
	case 1:
		w.Pong = &Pong{}
		return w.Pong.ReadFrom(I)
	case 2:
		w.InvalidSignature = true
	case 3:
		w.Unauthorized = true
	case 4:
		w.RequestQueued = true
	case 5:
		w.TryAgainLater = true
	case 6:
		w.AgentRegistered = &AgentRegistered{}
		return w.AgentRegistered.ReadFrom(I)
	case 7:
		w.AgentPortMapping = &AgentPortMapping{}
		return w.AgentPortMapping.ReadFrom(I)
	case 8:
		w.UdpChannelDetails = &UdpChannelDetails{}
		return w.UdpChannelDetails.ReadFrom(I)
	}
	return fmt.Errorf("invalid ControlResponse id")
}
