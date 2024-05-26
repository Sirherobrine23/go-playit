package enc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

func ReadU8(r io.Reader) uint8 {
	var d uint8
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func ReadU16(r io.Reader) uint16 {
	var d uint16
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func ReadU32(r io.Reader) uint32 {
	var d uint32
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func ReadU64(r io.Reader) uint64 {
	var d uint64
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}

func WriteU8(w io.Writer, d uint8) error {
	return binary.Write(w, binary.BigEndian, d)
}
func WriteU16(w io.Writer, d uint16) error {
	return binary.Write(w, binary.BigEndian, d)
}
func WriteU32(w io.Writer, d uint32) error {
	return binary.Write(w, binary.BigEndian, d)
}
func WriteU64(w io.Writer, d uint64) error {
	return binary.Write(w, binary.BigEndian, d)
}

func Read8(r io.Reader) int8 {
	var d int8
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func Read16(r io.Reader) int16 {
	var d int16
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func Read32(r io.Reader) int32 {
	var d int32
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func Read64(r io.Reader) int64 {
	var d int64
	err := binary.Read(r, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}
	return d
}
func Write8(w io.Writer, d int8) error {
	return binary.Write(w, binary.BigEndian, &d)
}
func Write16(w io.Writer, d int16) error {
	return binary.Write(w, binary.BigEndian, &d)
}
func Write32(w io.Writer, d int32) error {
	return binary.Write(w, binary.BigEndian, &d)
}
func Write64(w io.Writer, d int64) error {
	return binary.Write(w, binary.BigEndian, &d)
}

func ReadByteN(r io.Reader, size int) (buff []byte, err error) {
	buff = make([]byte, size)
	for index := range buff {
		if err = binary.Read(r, binary.BigEndian, &buff[index]); err != nil {
			buff = buff[:index]
			return
		}
	}
	return
}
func WriteBytes(w io.Writer, buff []byte) error {
	return binary.Write(w, binary.BigEndian, &buff)
}

func AddrWrite(w io.Writer, addr netip.Addr) error {
	if addr.Is6() {
		if err := WriteU8(w, 6); err != nil {
			return err
		} else if err = WriteBytes(w, addr.AsSlice()); err != nil {
			return err
		}
		return nil
	}
	if err := WriteU8(w, 4); err != nil {
		return err
	} else if err = WriteBytes(w, addr.AsSlice()); err != nil {
		return err
	}
	return nil
}
func AddrRead(r io.Reader) (addr netip.Addr, err error) {
	var buff []byte
	switch ReadU8(r) {
	case 4:
		if buff, err = ReadByteN(r, 4); err != nil {
			return
		}
		addr = netip.AddrFrom4([4]byte(buff))
		return
	case 6:
		if buff, err = ReadByteN(r, 16); err != nil {
			return
		}
		netip.AddrFrom16([16]byte(buff))
		return
	}
	err = fmt.Errorf("connet get ip type")
	return
}

func AddrPortRead(r io.Reader) (netip.AddrPort, error) {
	switch ReadU8(r) {
	case 4:
		buff, err := ReadByteN(r, 4)
		if err != nil {
			return netip.AddrPort{}, err
		}
		return netip.AddrPortFrom(netip.AddrFrom4([4]byte(buff)), ReadU16(r)), nil
	case 6:
		buff, err := ReadByteN(r, 16)
		if err != nil {
			return netip.AddrPort{}, err
		}
		return netip.AddrPortFrom(netip.AddrFrom16([16]byte(buff)), ReadU16(r)), nil
	}
	return netip.AddrPort{}, fmt.Errorf("connet get ip type")
}
func AddrPortWrite(w io.Writer, addr netip.AddrPort) error {
	if !addr.IsValid() {
		return fmt.Errorf("invalid ip address")
	} else if addr.Addr().Is6() {
		if err := WriteU8(w, 6); err != nil {
			return err
		} else if err = binary.Write(w, binary.BigEndian, addr.Addr().AsSlice()); err != nil {
			return err
		}
		return nil
	}
	if err := WriteU8(w, 4); err != nil {
		return err
	} else if err = binary.Write(w, binary.BigEndian, addr.Addr().AsSlice()); err != nil {
		return err
	}
	return nil
}

func ReadOption(r io.Reader, callback func(r io.Reader) (err error)) error {
	switch ReadU8(r) {
	case 0:
		return nil
	case 1:
		return callback(r)
	}
	return fmt.Errorf("invalid Option value")
}
