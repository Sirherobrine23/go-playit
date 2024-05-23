package tunnel

import (
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"time"
)

type ControlRequest struct {
	Ping                  *Ping
	AgentRegister         *AgentRegister
	AgentKeepAlive        *AgentSessionId
	SetupUdpChannel       *AgentSessionId
	AgentCheckPortMapping *AgentCheckPortMapping
}

func (Control *ControlRequest) WriteTo(w io.Writer) (n int64, err error) {
	if Control.Ping != nil {
		n, err = writeU32(w, 6)
		if err != nil {
			return
		}
		n2 := n
		n, err = Control.Ping.WriteTo(w)
		if err != nil {
			return n2, err
		}
		n += n2
		return
	} else if Control.AgentRegister != nil {
		n, err = writeU32(w, 2)
		if err != nil {
			return
		}
		n2 := n
		n, err = Control.AgentRegister.WriteTo(w)
		if err != nil {
			return n2, err
		}
		n += n2
		return
	} else if Control.AgentKeepAlive != nil {
		n, err = writeU32(w, 3)
		if err != nil {
			return
		}
		n2 := n
		n, err = Control.AgentKeepAlive.WriteTo(w)
		if err != nil {
			return n2, err
		}
		n += n2
		return
	} else if Control.SetupUdpChannel != nil {
		n, err = writeU32(w, 4)
		if err != nil {
			return
		}
		n2 := n
		n, err = Control.SetupUdpChannel.WriteTo(w)
		if err != nil {
			return n2, err
		}
		n += n2
		return
	} else if Control.AgentCheckPortMapping != nil {
		n, err = writeU32(w, 5)
		if err != nil {
			return
		}
		n2 := n
		n, err = Control.AgentCheckPortMapping.WriteTo(w)
		if err != nil {
			return n2, err
		}
		n += n2
		return
	}
	err = fmt.Errorf("set ControlRequest")
	return
}
func (Control *ControlRequest) ReadFrom(r io.Reader) (n int64, err error) {
	n = 1
	switch readU32(r) {
	case 1:
		Control.Ping = new(Ping)
		np, err := Control.Ping.ReadFrom(r)
		return np + n, err
	case 2:
		Control.AgentRegister = new(AgentRegister)
		np, err := Control.AgentRegister.ReadFrom(r)
		return np + n, err
	case 3:
		Control.AgentKeepAlive = new(AgentSessionId)
		np, err := Control.AgentKeepAlive.ReadFrom(r)
		return np + n, err
	case 4:
		Control.SetupUdpChannel = new(AgentSessionId)
		np, err := Control.SetupUdpChannel.ReadFrom(r)
		return np + n, err
	case 5:
		Control.AgentCheckPortMapping = new(AgentCheckPortMapping)
		np, err := Control.AgentCheckPortMapping.ReadFrom(r)
		return np + n, err
	}
	err = fmt.Errorf("invalid ControlRequest id")
	return
}

type AgentCheckPortMapping struct {
	AgentSessionId AgentSessionId
	PortRange      PortRange
}

func (Agent *AgentCheckPortMapping) WriteTo(w io.Writer) (n int64, err error) {
	n, err = Agent.AgentSessionId.WriteTo(w)
	if err != nil {
		return
	}
	n2 := n
	n, err = Agent.PortRange.WriteTo(w)
	return n + n2, err
}
func (Agent *AgentCheckPortMapping) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = Agent.AgentSessionId.ReadFrom(r)
	if err != nil {
		return
	}
	n2 := n
	n, err = Agent.AgentSessionId.ReadFrom(r)
	return n + n2, err
}

type Ping struct {
	Now         time.Time
	CurrentPing *time.Time
	SessionId   *AgentSessionId
}

func (ping *Ping) WriteTo(w io.Writer) (n int64, err error) {
	n, err = writeU64(w, uint64(ping.Now.UnixMilli()))
	if err != nil {
		return
	}
	n2 := n
	if n, err = writeOption(w, ping.CurrentPing, func(w io.Writer) (int64, error) {
		return writeU64(w, uint64(ping.CurrentPing.UnixMilli()))
	}); err != nil {
		n = n2
		return
	}
	n += n2
	if n, err = writeOption(w, ping.SessionId, ping.SessionId.WriteTo); err != nil {
		n = n2
		return
	}
	return
}
func (ping *Ping) ReadFrom(r io.Reader) (n int64, err error) {
	ping.Now = time.UnixMilli(int64(readU64(r)))
	n, err = readOption(r, func(r io.Reader) (n int64, err error) {
		ping.CurrentPing = new(time.Time)
		d, _ := time.UnixMilli(int64(readU64(r))).MarshalBinary()
		ping.CurrentPing.UnmarshalBinary(d)
		return 8, nil
	})
	n += 8
	if err != nil {
		return
	}
	n, err = readOption(r, func(r io.Reader) (n int64, err error) {
		return ping.SessionId.ReadFrom(r)
	})
	if err != nil {
		return
	}
	return
}

type AgentRegister struct {
	AccountID, AgentId, AgentVersion, Timestamp uint64
	ClientAddr, TunnelAddr                      netip.AddrPort
	Signature                                   [32]byte
}

func (agent *AgentRegister) writePlain() *bytes.Buffer {
	buff := new(bytes.Buffer)
	writeU64(buff, agent.AccountID)
	writeU64(buff, agent.AgentId)
	writeU64(buff, agent.AgentVersion)
	writeU64(buff, agent.Timestamp)
	addrPortWrite(buff, agent.ClientAddr)
	addrPortWrite(buff, agent.TunnelAddr)
	return buff
}
func (agent *AgentRegister) UpdateSignature(hmac HmacSha256) {
	agent.Signature = hmac.Sign(agent.writePlain().Bytes())
}
func (agent *AgentRegister) VerifySignature(hmac HmacSha256) bool {
	return hmac.Verify(agent.writePlain().Bytes(), agent.Signature[:])
}

func (AgentReg *AgentRegister) WriteTo(w io.Writer) (n int64, err error) {
	if _, err := writeU64(w, AgentReg.AccountID); err != nil {
		return 0, err
	} else if _, err := writeU64(w, AgentReg.AgentId); err != nil {
		return 0, err
	} else if _, err := writeU64(w, AgentReg.AgentVersion); err != nil {
		return 0, err
	} else if _, err := writeU64(w, AgentReg.Timestamp); err != nil {
		return 0, err
	}
	n = 8 * 4
	if n2, err := addrPortWrite(w, AgentReg.ClientAddr); err != nil {
		return n, err
	} else if n3, err := addrPortWrite(w, AgentReg.TunnelAddr); err != nil {
		return n + n2, err
	} else {
		n += n3
	}
	if n4, err := w.Write(AgentReg.Signature[:]); err != nil {
		return n, err
	} else {
		n += int64(n4)
	}
	return
}
func (AgentReg *AgentRegister) ReadFrom(r io.Reader) (n int64, err error) {
	AgentReg.AccountID, AgentReg.AccountID, AgentReg.AgentVersion, AgentReg.Timestamp = readU64(r), readU64(r), readU64(r), readU64(r)
	if AgentReg.ClientAddr, n, err = addrPortRead(r); err != nil {
		return
	} else if AgentReg.TunnelAddr, n, err = addrPortRead(r); err != nil {
		return
	}
	AgentReg.Signature = [32]byte(make([]byte, 32))
	if n2, _ := r.Read(AgentReg.Signature[:]); n != 32 {
		return int64(n2), fmt.Errorf("missing signature")
	}
	return
}

type ControlResponse struct {
	InvalidSignature, Unauthorized, RequestQueued, TryAgainLater bool
	Pong                                                         *Pong
	AgentRegistered                                              *AgentRegistered
	AgentPortMapping                                             *AgentPortMapping
	UdpChannelDetails                                            *UdpChannelDetails
}

func (Control *ControlResponse) WriteTo(w io.Writer) (n int64, err error) {
	defer func() {
		if err == nil {
			n += 4
		}
	}()
	if Control.Pong != nil {
		writeU32(w, 1)
		n, err = Control.Pong.WriteTo(w)
		return
	} else if Control.InvalidSignature {
		return writeU32(w, 2)
	} else if Control.Unauthorized {
		return writeU32(w, 3)
	} else if Control.RequestQueued {
		return writeU32(w, 4)
	} else if Control.TryAgainLater {
		return writeU32(w, 5)
	} else if Control.AgentRegistered != nil {
		writeU32(w, 6)
		return Control.AgentRegistered.WriteTo(w)
	} else if Control.AgentPortMapping != nil {
		writeU32(w, 7)
		return Control.AgentPortMapping.WriteTo(w)
	} else if Control.UdpChannelDetails != nil {
		writeU32(w, 8)
		return Control.UdpChannelDetails.WriteTo(w)
	} else {
		err = fmt.Errorf("insert any options to write")
	}
	return
}
func (Control *ControlResponse) ReadFrom(r io.Reader) (n int64, err error) {
	defer func() {
		if err == nil {
			n += 4
		}
	}()
	switch readU32(r) {
	case 1:
		Control.Pong = &Pong{}
		return Control.Pong.ReadFrom(r)
	case 2:
		Control.InvalidSignature = true
		return
	case 3:
		Control.Unauthorized = true
		return
	case 4:
		Control.RequestQueued = true
		return
	case 5:
		Control.TryAgainLater = true
		return
	case 6:
		Control.AgentRegistered = &AgentRegistered{}
		return Control.AgentRegistered.ReadFrom(r)
	case 7:
		Control.AgentPortMapping = &AgentPortMapping{}
		return Control.AgentPortMapping.ReadFrom(r)
	case 8:
		Control.UdpChannelDetails = &UdpChannelDetails{}
		return Control.UdpChannelDetails.ReadFrom(r)
	default:
		err = fmt.Errorf("invalid ControlResponse id")
	}
	return
}

type AgentPortMapping struct {
	Range PortRange
	Found *AgentPortMappingFound
}

func (Agent *AgentPortMapping) WriteTo(w io.Writer) (n int64, err error) {
	Agent.Range.WriteTo(w)
	Agent.Found.WriteTo(w)
	return
}
func (Agent *AgentPortMapping) ReadFrom(r io.Reader) (n int64, err error) {
	Agent.Range.ReadFrom(r)
	Agent.Found.ReadFrom(r)
	return
}

type AgentPortMappingFound struct {
	ToAgent *AgentSessionId
}

func (Agent *AgentPortMappingFound) WriteTo(w io.Writer) (n int64, err error) {
	if Agent.ToAgent != nil {
		writeU32(w, 1)
		Agent.ToAgent.WriteTo(w)
		return
	}
	return
}
func (Agent *AgentPortMappingFound) ReadFrom(r io.Reader) (n int64, err error) {
	if readU32(r) == 1 {
		defer func() { n += 4 }()
		Agent.ToAgent = new(AgentSessionId)
		return Agent.ToAgent.ReadFrom(r)
	}
	return 4, fmt.Errorf("unknown AgentPortMappingFound id")
}

type UdpChannelDetails struct {
	TunnelAddr netip.AddrPort
	Token      []byte
}

func (UdpChannel *UdpChannelDetails) WriteTo(w io.Writer) (n int64, err error)  {
	addrPortWrite(w, UdpChannel.TunnelAddr)
	writeU64(w, uint64(len(UdpChannel.Token)))
	writeBytes(w, UdpChannel.Token)
	return
}
func (UdpChannel *UdpChannelDetails) ReadFrom(r io.Reader) (n int64, err error) {
	UdpChannel.TunnelAddr, _, _ = addrPortRead(r)
	UdpChannel.Token, _ = readByteN(r, int(readU64(r)))
	return
}

type Pong struct {
	RequestNow, ServerNow  time.Time
	ServerId               uint64
	DataCenterId           uint32
	ClientAddr, TunnelAddr netip.AddrPort
	SessionExpireAt        *time.Time
}

func (pong *Pong) WriteTo(w io.Writer) (n int64, err error)  {
	
}
func (pong *Pong) ReadFrom(r io.Reader) (n int64, err error) {}

type AgentRegistered struct {
	Id        AgentSessionId
	ExpiresAt time.Time
}

func (agent *AgentRegistered) WriteTo(w io.Writer) (n int64, err error)  {}
func (agent *AgentRegistered) ReadFrom(r io.Reader) (n int64, err error) {}
