package tunnel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type ConnectedControl struct {
	ControlAddr netip.AddrPort
	Udp         net.Conn
	Pong        Pong
}

func (Control *ConnectedControl) Authenticate(Api api.Api) (*AuthenticatedControl, error) {
	if !Control.Pong.ClientAddr.AddrPort.IsValid() {
		return nil, fmt.Errorf("invalid pong Client address")
	} else if !Control.Pong.TunnelAddr.AddrPort.IsValid() {
		return nil, fmt.Errorf("invalid pong Tunnel address")
	}

	tk, err := Api.ProtoRegisterRegister(Control.Pong.ClientAddr.AddrPort, Control.Pong.TunnelAddr.AddrPort)
	if err != nil {
		return nil, err
	}

	tkBytes, err := hex.DecodeString(tk)
	if err != nil {
		return nil, err
	}

	for tr := 3; tr > 0; tr-- {
		buffer := bytes.NewBuffer([]byte{})
		if err := (&ControlRpcMessage[*RawSlice]{
			RequestID: 10,
			Content: &RawSlice{
				Buff: tkBytes,
			},
		}).WriteTo(buffer); err != nil {
			return nil, err
		}

		_, err := Control.Udp.Write(buffer.Bytes())
		if err != nil {
			return nil, err
		}

		reciver := make([]byte, 1024)
		Control.Udp.SetReadDeadline(time.Now().Add(time.Second * 5))
		if _, err = Control.Udp.Read(reciver); err != nil {
			return nil, err
		}

		feed := &ControlFeed{}
		if err = feed.ReadFrom(bytes.NewReader(reciver)); err != nil {
			return nil, err
		} else if feed.Response == nil || feed.Response.Content == nil {
			return nil, fmt.Errorf("cannot get response")
		}

		controlRes := feed.Response.Content
		if controlRes.RequestQueued {
			time.Sleep(time.Second)
		} else if controlRes.InvalidSignature {
			return nil, fmt.Errorf("register return invalid signature")
		} else if controlRes.Unauthorized {
			return nil, fmt.Errorf("unauthorized")
		} else if controlRes.AgentRegistered != nil {
			auth := AuthenticatedControl{
				ApiClient:   Api,
				Conn:        *Control,
				LastPong:    Control.Pong,
				CurrentPing: nil,
				Registered:  *controlRes.AgentRegistered,
				Buff:        []byte{},
				ForceEpired: false,
			}
			return &auth, nil
		}
	}
	return nil, fmt.Errorf("expected AgentRegistered but got something else")
}
