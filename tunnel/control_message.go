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
	w.TunnelAddr = AddressPort{}

	if err := w.ClientAddr.ReadFrom(I); err != nil {
		return err
	} else if err := w.TunnelAddr.ReadFrom(I); err != nil {
		return err
	}

	Sess := ReadU64(I)
	w.SessionExpireAt = &Sess
	return nil
}

type AgentRegister struct {
	AccountID, AgentId, AgentVersion, Timestamp uint64
	ClientAddr, TunnelAddr                      AddressPort
	Signature                                   []byte // 32 bytes
}

func (w *AgentRegister) WritePlain(buff io.Writer) error {
	if err := WriteU64(buff, w.AccountID); err != nil {
		return err
	} else if err := WriteU64(buff, w.AgentId); err != nil {
		return err
	} else if err := WriteU64(buff, w.AgentVersion); err != nil {
		return err
	} else if err := WriteU64(buff, w.Timestamp); err != nil {
		return err
	} else if err := w.ClientAddr.WriteTo(buff); err != nil {
		return err
	} else if err := w.TunnelAddr.WriteTo(buff); err != nil {
		return err
	}
	return nil
}
func (w *AgentRegister) WriteTo(I io.Writer) error {
	if err := WriteU64(I, w.AccountID); err != nil {
		return err
	} else if err := WriteU64(I, w.AgentId); err != nil {
		return err
	} else if err := WriteU64(I, w.AgentVersion); err != nil {
		return err
	} else if err := WriteU64(I, w.Timestamp); err != nil {
		return err
	} else if err := w.ClientAddr.WriteTo(I); err != nil {
		return err
	} else if err := w.TunnelAddr.WriteTo(I); err != nil {
		return err
	} else if err := binary.Write(I, binary.BigEndian, w.Signature); err != nil {
		return err
	}
	return nil
}
func (w *AgentRegister) ReadFrom(I io.Reader) error {
	w.AccountID = ReadU64(I)
	w.AgentId = ReadU64(I)
	w.AgentVersion = ReadU64(I)
	w.Timestamp = ReadU64(I)
	w.ClientAddr, w.TunnelAddr = AddressPort{}, AddressPort{}
	if err := w.ClientAddr.ReadFrom(I); err != nil {
		return err
	} else if err := w.TunnelAddr.ReadFrom(I); err != nil {
		return err
	}
	w.Signature = make([]byte, 32)
	if err := ReadBuff(I, w.Signature); err != nil {
		return err
	}
	return nil
}

type AgentCheckPortMapping struct {
	AgentSessionId AgentSessionId
	PortRange      PortRange
}

func (w *AgentCheckPortMapping) WriteTo(I io.Writer) error {
	if err := w.AgentSessionId.WriteTo(I); err != nil {
		return err
	} else if err := w.PortRange.WriteTo(I); err != nil {
		return err
	}
	return nil
}
func (w *AgentCheckPortMapping) ReadFrom(I io.Reader) error {
	w.AgentSessionId, w.PortRange = AgentSessionId{}, PortRange{}
	if err := w.AgentSessionId.ReadFrom(I); err != nil {
		return err
	} else if err := w.PortRange.ReadFrom(I); err != nil {
		return err
	}
	return nil
}

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

type AgentRegistered struct {
	ID        AgentSessionId
	ExpiresAt time.Time
}

func (w *AgentRegistered) WriteTo(I io.Writer) error {
	if err := w.ID.WriteTo(I); err != nil {
		return err
	} else if err := WriteU64(I, uint64(w.ExpiresAt.UnixMilli())); err != nil {
		return err
	}
	return nil
}
func (w *AgentRegistered) ReadFrom(I io.Reader) error {
	w.ID = AgentSessionId{}
	if err := w.ID.ReadFrom(I); err != nil {
		return err
	}
	w.ExpiresAt = time.UnixMilli(int64(ReadU64(I)))
	return nil
}

type AgentPortMappingFound struct {
	ToAgent *AgentSessionId
}

func (agentPort *AgentPortMappingFound) WriteTo(I io.Writer) error {
	if agentPort.ToAgent != nil {
		if err := WriteU32(I, 1); err != nil {
			return err
		} else if err := agentPort.ToAgent.WriteTo(I); err != nil {
			return err
		}
	}
	return nil
}
func (agentPort *AgentPortMappingFound) ReadFrom(I io.Reader) error {
	if ReadU32(I) == 1 {
		agentPort.ToAgent = &AgentSessionId{}
		return agentPort.ToAgent.ReadFrom(I)
	}
	return fmt.Errorf("unknown AgentPortMappingFound id")
}

type AgentPortMapping struct {
	Range PortRange
	Found *AgentPortMappingFound
}

func (w *AgentPortMapping) WriteTo(I io.Writer) error {
	if err := w.Range.WriteTo(I); err != nil {
		return err
	} else if err := w.Found.WriteTo(I); err != nil {
		return err
	}
	return nil
}
func (w *AgentPortMapping) ReadFrom(I io.Reader) error {
	if err := w.Range.ReadFrom(I); err != nil {
		return err
	} else if err := w.Found.ReadFrom(I); err != nil {
		return err
	}
	return nil
}

type UdpChannelDetails struct {
	TunnelAddr AddressPort
	Token      []byte
}

func (w *UdpChannelDetails) WriteTo(I io.Writer) error {
	if err := w.TunnelAddr.WriteTo(I); err != nil {
		return err
	} else if err := WriteU64(I, uint64(len(w.Token))); err != nil {
		return err
	} else if err := binary.Write(I, binary.BigEndian, w.Token); err != nil {
		return err
	}
	return nil
}
func (w *UdpChannelDetails) ReadFrom(I io.Reader) error {
	w.TunnelAddr = AddressPort{}
	if err := w.TunnelAddr.ReadFrom(I); err != nil {
		return err
	}
	w.Token = make([]byte, ReadU64(I))
	if err := ReadBuff(I, w.Token); err != nil {
		return err
	}
	return nil
}

type ControlResponse struct {
	InvalidSignature  bool
	Unauthorized      bool
	RequestQueued     bool
	TryAgainLater     bool
	Pong              *Pong
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
	return fmt.Errorf("set one option to write")
}
func (w *ControlResponse) ReadFrom(I io.Reader) error {
	switch ReadU32(I) {
	case 1:
		w.Pong = &Pong{}
		return w.Pong.ReadFrom(I)
	case 2:
		w.InvalidSignature = true
		return nil
	case 3:
		w.Unauthorized = true
		return nil
	case 4:
		w.RequestQueued = true
		return nil
	case 5:
		w.TryAgainLater = true
		return nil
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
