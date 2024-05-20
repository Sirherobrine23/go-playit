package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

type RawSlice struct {
	Buff []byte
}

func (w *RawSlice) WriteTo(I io.Writer) error {
	_, err := I.Write(w.Buff)
	return err
}
func (w *RawSlice) ReadFrom(I io.Reader) error {
	_, err := I.Read(w.Buff)
	return err
}

func ReadU8(w io.Reader) uint8 {
	var value uint8
	binary.Read(w, binary.BigEndian, &value)
	return value
}

func WriteU8(w io.Writer, value uint8) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadU16(w io.Reader) uint16 {
	var value uint16
	binary.Read(w, binary.BigEndian, &value)
	return value
}

func WriteU16(w io.Writer, value uint16) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadU32(w io.Reader) uint32 {
	var value uint32
	binary.Read(w, binary.BigEndian, &value)
	return value
}

func WriteU32(w io.Writer, value uint32) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadU64(w io.Reader) uint64 {
	var value uint64
	binary.Read(w, binary.BigEndian, &value)
	return value
}

func WriteU64(w io.Writer, value uint64) error {
	return binary.Write(w, binary.BigEndian, value)
}

func ReadBuff(w io.Reader, buff []byte) error {
	for index := range buff {
		if err := binary.Read(w, binary.BigEndian, &buff[index]); err != nil {
			return err
		}
	}
	return nil
}

func ReadBuffN(w io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	return buff, ReadBuff(w, buff)
}

func ReadU8Buff(w io.Reader, buff []uint8) error {
	for index := range buff {
		if err := binary.Read(w, binary.BigEndian, &buff[index]); err != nil {
			return err
		}
	}
	return nil
}

func ReadU16Buff(w io.Reader, buff []uint16) error {
	for index := range buff {
		if err := binary.Read(w, binary.BigEndian, &buff[index]); err != nil {
			return err
		}
	}
	return nil
}

func ReadU32Buff(w io.Reader, buff []uint32) error {
	for index := range buff {
		if err := binary.Read(w, binary.BigEndian, &buff[index]); err != nil {
			return err
		}
	}
	return nil
}

func ReadU64Buff(w io.Reader, buff []uint64) error {
	for index := range buff {
		if err := binary.Read(w, binary.BigEndian, &buff[index]); err != nil {
			return err
		}
	}
	return nil
}

func ReadOption(w io.Reader, callback func(reader io.Reader) error) error {
	code := ReadU8(w)
	if code == 1 {
		return callback(w)
	}
	return nil
}

func WriteOption(w io.Writer, value MessageEncoding) error {
	fmt.Printf("%+v\n", value)
	if value != nil {
		if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
			return err
		}
		return value.WriteTo(w)
	}
	return binary.Write(w, binary.BigEndian, uint8(0))
}

func WriteOptionU8(w io.Writer, value *uint8) error {
	if value == nil {
		return binary.Write(w, binary.BigEndian, uint8(0))
	}
	if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
		return err
	}
	return WriteU8(w, *value)
}

func WriteOptionU16(w io.Writer, value *uint16) error {
	if value == nil {
		return binary.Write(w, binary.BigEndian, uint8(0))
	}
	if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
		return err
	}
	return WriteU16(w, *value)
}

func WriteOptionU32(w io.Writer, value *uint32) error {
	if value == nil {
		return binary.Write(w, binary.BigEndian, uint8(0))
	}
	if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
		return err
	}
	return WriteU32(w, *value)
}

func WriteOptionU64(w io.Writer, value *uint64) error {
	if value == nil {
		return binary.Write(w, binary.BigEndian, uint8(0))
	}
	if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
		return err
	}
	return WriteU64(w, *value)
}

type AddressPort struct {
	netip.AddrPort
}

func (sock *AddressPort) WriteTo(w io.Writer) error {
	addr := sock.Addr()
	ip, _ := addr.MarshalBinary()
	if addr.Is6() {
		if err := WriteU8(w, uint8(6)); err != nil {
			return err
		} else if _, err = w.Write(ip); err != nil {
			return err
		}
	} else {
		if err := WriteU8(w, uint8(4)); err != nil {
			return err
		} else if _, err = w.Write(ip); err != nil {
			return err
		}
	}
	if err := WriteU16(w, sock.Port()); err != nil {
		return err
	}
	return nil
}
func (sock *AddressPort) ReadFrom(w io.Reader) error {
	switch ReadU8(w) {
	case 4:
		buff, err := ReadBuffN(w, 4)
		if err != nil {
			return err
		}
		sock.AddrPort = netip.AddrPortFrom(netip.AddrFrom4([4]byte(buff)), ReadU16(w))
		return nil
	case 6:
		buff, err := ReadBuffN(w, 16)
		if err != nil {
			return err
		}
		sock.AddrPort = netip.AddrPortFrom(netip.AddrFrom16([16]byte(buff)), ReadU16(w))
		return nil
	}
	return fmt.Errorf("cannot get IP type")
}
