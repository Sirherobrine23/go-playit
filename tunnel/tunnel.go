package tunnel

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
)

type Tunnel struct {
	ApiClaim api.Claim
	Clients  map[any]net.Conn
}

type ConnectedControl struct {}

func (Tun *Tunnel) Setup() (net.Conn, error) {
	controls, err := api.AgentRoutings(Tun.ApiClaim.Secret, nil)
	if err != nil {
		return nil, err
	}

	IsIPv6 := func(str string) bool {
		ip := net.ParseIP(str)
		return ip != nil && strings.Contains(str, ":")
	}

	for _, Addr := range append(controls.Targets6, controls.Targets4...) {
		fmt.Println(Addr.String())
		connAddress := "%s:5525"
		if IsIPv6(Addr.String()) {
			connAddress = "[%s]:5525"
		}

		conn, err := net.Dial("udp", fmt.Sprintf(connAddress, Addr.String()))
		if err != nil {
			continue
		}

		buffer := bytes.NewBuffer([]byte{})
		err = (&ControlRpcMessage{
			RequestID: 1,
			Content: &ControlRequest{
				Ping: &Ping{
					Now:         time.Now(),
					CurrentPing: nil,
					SessionID:   nil,
				},
			},
		}).WriteTo(buffer)
		if err != nil {
			conn.Close()
			return nil, err
		}
		fmt.Printf("Rpc Go, Size: %d, Data: %+v\n", len(buffer.Bytes()), buffer.Bytes())

		_, err = buffer.WriteTo(conn)
		if err != nil {
			conn.Close()
			return nil, err
		}

		buffer.Reset()
		conn.SetReadDeadline(time.Now().Add(time.Second * 5))

		bytesRead := make([]byte, 2048)
		_, err = conn.Read(bytesRead)
		if err != nil {
			conn.Close()
			return nil, err
		}
		fmt.Printf("Rpl Go, Size: %d, Data: %+v\n", len(bytesRead), bytesRead)

		reader := bytes.NewReader(bytesRead)
		res := ControlFeed{}
		if err := res.ReadFrom(reader); err != nil {
			return nil, err
		} else if res.Response == nil || res.Response.RequestID != 1 {
			conn.Close()
			return nil, fmt.Errorf("got response with unexpected request_id")
		}

		Response, isResponse := res.Response.Content.(*ControlResponse)
		if !isResponse {
			conn.Close()
			return nil, fmt.Errorf("expected controlRequest")
		}

		if Response.Pong == nil {
			conn.Close()
			return nil, fmt.Errorf("expected pong got other response")
		}

		return conn, err
	}

	return nil, nil
}
