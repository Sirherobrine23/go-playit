package tunnel

import (
	"bytes"
	"encoding/binary"
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

func (Tun *Tunnel) Setup() (net.Conn, error) {
	controls, err := api.AgentRoutings(Tun.ApiClaim.Secret, nil)
	if err != nil {
		return nil, err
	}

	IsIPv6 := func (str string) bool {
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

		buffer := bytes.NewBuffer([]byte{1})
		binary.Write(buffer, binary.BigEndian, time.Now().UnixMilli())
		binary.Write(buffer, binary.BigEndian, int8(0))
		binary.Write(buffer, binary.BigEndian, int8(0))

		fmt.Println(buffer.Bytes())
		size, err := buffer.WriteTo(conn)
		buffer.Reset()

		if err != nil {
			conn.Close()
			continue // Skip connetion
		} else if size > 0 {
			var size int
			buffer := make([]byte, 2048)
			for l := 5; l != 0; l-- {
				conn.SetReadDeadline(time.Now().Add(time.Millisecond * 5))
				size, err = conn.Read(buffer)
				if err != nil {
					continue
				}

				fmt.Println(size)
				fmt.Println(buffer)

			}
			return conn, err
		}
	}

	return nil, nil
}