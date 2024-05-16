package tunnel

import (
	"fmt"
	"net"
	"time"
	"bytes"
	"encoding/binary"
)

var (
	ControlHost string = "control.playit.gg"
	ControlPort int    = 5525
)

func Connect() error {
	dial, err := net.Dial("udp", fmt.Sprintf("%s:%d", ControlHost, ControlPort))
	if err != nil {
		return err
	}
	dial.SetDeadline(time.Now().Add(time.Duration(time.Second * 3)))

	buff := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buff, binary.BigEndian, int32(1))
	binary.Write(buff, binary.BigEndian, int32(1))
	binary.Write(buff, binary.BigEndian, int32(0))
	buff.Write([]byte{0})

	fmt.Printf("%+v\n", buff.Bytes())
	_, err = dial.Write(buff.Bytes())
	if err != nil {
		return err
	}

	rec := make([]byte, 16)
	dial.Read(rec)
	fmt.Printf("%+v\n", rec)
	buff = bytes.NewBuffer(rec)

	var value int64
	binary.Read(buff, binary.BigEndian, &value)
	fmt.Printf("%d\n", value)

	return nil
}
