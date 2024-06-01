package proto

import (
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

type ControlRequest struct {
	Ping                  *Ping
	AgentRegister         *AgentRegister
	AgentKeepAlive        *AgentSessionId
	SetupUdpChannel       *AgentSessionId
	AgentCheckPortMapping *AgentCheckPortMapping
}

func (Control *ControlRequest) WriteTo(w io.Writer) error {
	defer debug.Printf("Write ControlRequest: %s\n", logfile.JSONString(Control))
	if Control.Ping != nil {
		if err := enc.WriteU32(w, 6); err != nil {
			debug.Printf("Write ControlRequest error: %s\n", err.Error())
			return err
		}
		return Control.Ping.WriteTo(w)
	} else if Control.AgentRegister != nil {
		if err := enc.WriteU32(w, 2); err != nil {
			debug.Printf("Write ControlRequest error: %s\n", err.Error())
			return err
		}
		return Control.AgentRegister.WriteTo(w)
	} else if Control.AgentKeepAlive != nil {
		if err := enc.WriteU32(w, 3); err != nil {
			debug.Printf("Write ControlRequest error: %s\n", err.Error())
			return err
		}
		return Control.AgentKeepAlive.WriteTo(w)
	} else if Control.SetupUdpChannel != nil {
		if err := enc.WriteU32(w, 4); err != nil {
			debug.Printf("Write ControlRequest error: %s\n", err.Error())
			return err
		}
		return Control.SetupUdpChannel.WriteTo(w)
	} else if Control.AgentCheckPortMapping != nil {
		if err := enc.WriteU32(w, 5); err != nil {
			debug.Printf("Write ControlRequest error: %s\n", err.Error())
			return err
		}
		return Control.AgentCheckPortMapping.WriteTo(w)
	}
	return fmt.Errorf("set ControlRequest")
}
func (Control *ControlRequest) ReadFrom(r io.Reader) (err error) {
	switch enc.ReadU32(r) {
	case 1:
		Control.Ping = new(Ping)
		err = Control.Ping.ReadFrom(r)
	case 2:
		Control.AgentRegister = new(AgentRegister)
		err = Control.AgentRegister.ReadFrom(r)
	case 3:
		Control.AgentKeepAlive = new(AgentSessionId)
		err = Control.AgentKeepAlive.ReadFrom(r)
	case 4:
		Control.SetupUdpChannel = new(AgentSessionId)
		err = Control.SetupUdpChannel.ReadFrom(r)
	case 5:
		Control.AgentCheckPortMapping = new(AgentCheckPortMapping)
		err = Control.AgentCheckPortMapping.ReadFrom(r)
	default:
		err = fmt.Errorf("invalid ControlRequest id")
	}
	debug.Printf("Read ControlRequest: %s\n", logfile.JSONString(Control))
	if err != nil {
		debug.Printf("Read ControlRequest error: %s\n", err.Error())
	}
	return
}

type AgentCheckPortMapping struct {
	AgentSessionId AgentSessionId
	PortRange      PortRange
}

func (Agent *AgentCheckPortMapping) WriteTo(w io.Writer) error {
	if err := Agent.AgentSessionId.WriteTo(w); err != nil {
		return err
	}
	return Agent.PortRange.WriteTo(w)
}
func (Agent *AgentCheckPortMapping) ReadFrom(r io.Reader) error {
	if err := Agent.AgentSessionId.ReadFrom(r); err != nil {
		return err
	}
	return Agent.AgentSessionId.ReadFrom(r)
}

type Ping struct {
	Now         time.Time
	CurrentPing *uint32
	SessionId   *AgentSessionId
}

func (ping *Ping) WriteTo(w io.Writer) error {
	if err := enc.WriteU64(w, uint64(ping.Now.UnixMilli())); err != nil {
		return err
	}

	if ping.CurrentPing == nil {
		if err := enc.WriteU8(w, 0); err != nil {
			return err
		}
	} else {
		if err := enc.WriteU8(w, 1); err != nil {
			return err
		} else if err := enc.WriteU32(w, *ping.CurrentPing); err != nil {
			return err
		}
	}

	if ping.SessionId == nil {
		if err := enc.WriteU8(w, 0); err != nil {
			return err
		}
	} else {
		if err := enc.WriteU8(w, 1); err != nil {
			return err
		} else if err := ping.SessionId.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}
func (ping *Ping) ReadFrom(r io.Reader) error {
	ping.Now = time.UnixMilli(int64(enc.ReadU64(r)))
	if err := enc.ReadOption(r, func(r io.Reader) error {
		*ping.CurrentPing = enc.ReadU32(r)
		return nil
	}); err != nil {
		return err
	}

	if err := enc.ReadOption(r, func(r io.Reader) error {
		return ping.SessionId.ReadFrom(r)
	}); err != nil {
		return err
	}
	return nil
}

type AgentRegister struct {
	AccountID, AgentId, AgentVersion, Timestamp uint64
	ClientAddr, TunnelAddr                      netip.AddrPort
	Signature                                   [32]byte
}

func (agent *AgentRegister) writePlain() *bytes.Buffer {
	buff := new(bytes.Buffer)
	enc.WriteU64(buff, agent.AccountID)
	enc.WriteU64(buff, agent.AgentId)
	enc.WriteU64(buff, agent.AgentVersion)
	enc.WriteU64(buff, agent.Timestamp)
	enc.AddrPortWrite(buff, agent.ClientAddr)
	enc.AddrPortWrite(buff, agent.TunnelAddr)
	return buff
}
func (agent *AgentRegister) UpdateSignature(hmac HmacSha256) {
	agent.Signature = hmac.Sign(agent.writePlain().Bytes())
}
func (agent *AgentRegister) VerifySignature(hmac HmacSha256) bool {
	return hmac.Verify(agent.writePlain().Bytes(), agent.Signature[:])
}

func (AgentReg *AgentRegister) WriteTo(w io.Writer) error {
	if err := enc.WriteU64(w, AgentReg.AccountID); err != nil {
		return err
	} else if err := enc.WriteU64(w, AgentReg.AgentId); err != nil {
		return err
	} else if err := enc.WriteU64(w, AgentReg.AgentVersion); err != nil {
		return err
	} else if err := enc.WriteU64(w, AgentReg.Timestamp); err != nil {
		return err
	}
	if err := enc.AddrPortWrite(w, AgentReg.ClientAddr); err != nil {
		return err
	} else if err := enc.AddrPortWrite(w, AgentReg.TunnelAddr); err != nil {
		return err
	}
	if _, err := w.Write(AgentReg.Signature[:]); err != nil {
		return err
	}
	return nil
}
func (AgentReg *AgentRegister) ReadFrom(r io.Reader) error {
	AgentReg.AccountID, AgentReg.AccountID, AgentReg.AgentVersion, AgentReg.Timestamp = enc.ReadU64(r), enc.ReadU64(r), enc.ReadU64(r), enc.ReadU64(r)
	var err error
	if AgentReg.ClientAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if AgentReg.TunnelAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	}
	AgentReg.Signature = [32]byte(make([]byte, 32))
	if n, _ := r.Read(AgentReg.Signature[:]); n != 32 {
		return fmt.Errorf("missing signature")
	}
	return nil
}

type ControlResponse struct {
	InvalidSignature, Unauthorized, RequestQueued, TryAgainLater bool
	Pong                                                         *Pong
	AgentRegistered                                              *AgentRegistered
	AgentPortMapping                                             *AgentPortMapping
	UdpChannelDetails                                            *UdpChannelDetails
}

func (Control *ControlResponse) WriteTo(w io.Writer) error {
	defer debug.Printf("Write Feed: %s\n", logfile.JSONString(&Control))
	if Control.Pong != nil {
		if err := enc.WriteU32(w, 1); err != nil {
			return err
		}
		return Control.Pong.WriteTo(w)
	} else if Control.InvalidSignature {
		return enc.WriteU32(w, 2)
	} else if Control.Unauthorized {
		return enc.WriteU32(w, 3)
	} else if Control.RequestQueued {
		return enc.WriteU32(w, 4)
	} else if Control.TryAgainLater {
		return enc.WriteU32(w, 5)
	} else if Control.AgentRegistered != nil {
		if err := enc.WriteU32(w, 6); err != nil {
			return err
		}
		return Control.AgentRegistered.WriteTo(w)
	} else if Control.AgentPortMapping != nil {
		if err := enc.WriteU32(w, 7); err != nil {
			return err
		}
		return Control.AgentPortMapping.WriteTo(w)
	} else if Control.UdpChannelDetails != nil {
		if err := enc.WriteU32(w, 8); err != nil {
			return err
		}
		return Control.UdpChannelDetails.WriteTo(w)
	}
	return fmt.Errorf("insert any options to write")
}
func (Control *ControlResponse) ReadFrom(r io.Reader) (err error) {
	code := enc.ReadU32(r)
	switch code {
	case 1:
		Control.Pong = new(Pong)
		err = Control.Pong.ReadFrom(r)
	case 2:
		Control.InvalidSignature = true
		err = nil
	case 3:
		Control.Unauthorized = true
		err = nil
	case 4:
		Control.RequestQueued = true
		err = nil
	case 5:
		Control.TryAgainLater = true
		err = nil
	case 6:
		Control.AgentRegistered = new(AgentRegistered)
		err = Control.AgentRegistered.ReadFrom(r)
	case 7:
		Control.AgentPortMapping = new(AgentPortMapping)
		err = Control.AgentPortMapping.ReadFrom(r)
	case 8:
		Control.UdpChannelDetails = new(UdpChannelDetails)
		err = Control.UdpChannelDetails.ReadFrom(r)
	default:
		err = fmt.Errorf("invalid ControlResponse id")
	}
	debug.Printf("Read ControlResponse (Code %d): %s\n", code, logfile.JSONString(Control))
	if err != nil {
		debug.Printf("Read ControlResponse (Code %d) error: %s\n", code, err.Error())
	}
	return
}

type AgentPortMapping struct {
	Range PortRange
	Found *AgentPortMappingFound
}

func (Agent *AgentPortMapping) WriteTo(w io.Writer) error {
	if err := Agent.Range.WriteTo(w); err != nil {
		return err
	} else if err := Agent.Found.WriteTo(w); err != nil {
		return err
	}
	return nil
}
func (Agent *AgentPortMapping) ReadFrom(r io.Reader) error {
	if err := Agent.Range.ReadFrom(r); err != nil {
		return err
	} else if err := Agent.Found.ReadFrom(r); err != nil {
		return err
	}
	return nil
}

type AgentPortMappingFound struct {
	ToAgent *AgentSessionId
}

func (Agent *AgentPortMappingFound) WriteTo(w io.Writer) error {
	if Agent.ToAgent != nil {
		if err := enc.WriteU32(w, 1); err != nil {
			return err
		} else if err := Agent.ToAgent.WriteTo(w); err != nil {
			return err
		}
		return nil
	}
	return nil
}
func (Agent *AgentPortMappingFound) ReadFrom(r io.Reader) error {
	if enc.ReadU32(r) == 1 {
		Agent.ToAgent = new(AgentSessionId)
		return Agent.ToAgent.ReadFrom(r)
	}
	return fmt.Errorf("unknown AgentPortMappingFound id")
}

type UdpChannelDetails struct {
	TunnelAddr netip.AddrPort
	Token      []byte
}

func (UdpChannel *UdpChannelDetails) WriteTo(w io.Writer) error {
	if err := enc.AddrPortWrite(w, UdpChannel.TunnelAddr); err != nil {
		return err
	} else if err := enc.WriteU64(w, uint64(len(UdpChannel.Token))); err != nil {
		return err
	} else if err := enc.WriteBytes(w, UdpChannel.Token); err != nil {
		return err
	}
	return nil
}
func (UdpChannel *UdpChannelDetails) ReadFrom(r io.Reader) error {
	var err error
	if UdpChannel.TunnelAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if UdpChannel.Token, err = enc.ReadByteN(r, int(enc.ReadU64(r))); err != nil {
		return err
	}
	return nil
}

type Pong struct {
	RequestNow, ServerNow  time.Time
	ServerId               uint64
	DataCenterId           uint32
	ClientAddr, TunnelAddr netip.AddrPort
	SessionExpireAt        *time.Time
}

func (pong *Pong) WriteTo(w io.Writer) error {
	if err := enc.Write64(w, pong.RequestNow.UnixMilli()); err != nil {
		return err
	} else if err := enc.Write64(w, pong.ServerNow.UnixMilli()); err != nil {
		return err
	} else if err := enc.WriteU64(w, pong.ServerId); err != nil {
		return err
	} else if err := enc.WriteU32(w, pong.DataCenterId); err != nil {
		return err
	} else if err := enc.AddrPortWrite(w, pong.ClientAddr); err != nil {
		return err
	} else if err := enc.AddrPortWrite(w, pong.TunnelAddr); err != nil {
		return err
	}

	if pong.SessionExpireAt == nil {
		if err := enc.Write8(w, 0); err != nil {
			return err
		}
	} else {
		if err := enc.Write8(w, 1); err != nil {
			return err
		} else if err := enc.Write64(w, pong.SessionExpireAt.UnixMilli()); err != nil {
			return err
		}
	}
	return nil
}
func (pong *Pong) ReadFrom(r io.Reader) error {
	pong.RequestNow = time.UnixMilli(enc.Read64(r))
	pong.ServerNow = time.UnixMilli(enc.Read64(r))
	pong.ServerId = enc.ReadU64(r)
	pong.DataCenterId = enc.ReadU32(r)
	var err error
	if pong.ClientAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if pong.TunnelAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if err := enc.ReadOption(r, func(r io.Reader) (err error) {
		pong.SessionExpireAt = new(time.Time)
		*pong.SessionExpireAt = time.UnixMilli(enc.Read64(r)) // Fix set SessionExpireAt
		return nil
	}); err != nil {
		return err
	}
	return nil
}

type AgentRegistered struct {
	Id        AgentSessionId
	ExpiresAt time.Time
}

func (agent *AgentRegistered) WriteTo(w io.Writer) error {
	if err := agent.Id.WriteTo(w); err != nil {
		return err
	} else if err := enc.Write64(w, agent.ExpiresAt.UnixMilli()); err != nil {
		return err
	}
	return nil
}
func (agent *AgentRegistered) ReadFrom(r io.Reader) error {
	if err := agent.Id.ReadFrom(r); err != nil {
		return err
	}
	agent.ExpiresAt = time.UnixMilli(enc.Read64(r))
	return nil
}
