package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

func readU8(r io.Reader) uint8 {
	var d uint8
	err := binary.Read(r, binary.BigEndian, d)
	if err != nil {
		panic(err)
	}
	return d
}
func readU16(r io.Reader) uint16 {
	var d uint16
	err := binary.Read(r, binary.BigEndian, d)
	if err != nil {
		panic(err)
	}
	return d
}
func readU32(r io.Reader) uint32 {
	var d uint32
	err := binary.Read(r, binary.BigEndian, d)
	if err != nil {
		panic(err)
	}
	return d
}
func readU64(r io.Reader) uint64 {
	var d uint64
	err := binary.Read(r, binary.BigEndian, d)
	if err != nil {
		panic(err)
	}
	return d
}

func writeU8(w io.Writer, d uint8) (int64, error) {
	return 1, binary.Write(w, binary.BigEndian, d)
}
func writeU16(w io.Writer, d uint16) (int64, error) {
	return 2, binary.Write(w, binary.BigEndian, d)
}
func writeU32(w io.Writer, d uint32) (int64, error) {
	return 4, binary.Write(w, binary.BigEndian, d)
}
func writeU64(w io.Writer, d uint64) (int64, error) {
	return 8, binary.Write(w, binary.BigEndian, d)
}

func readByteN(r io.Reader, size int) (buff []byte, err error) {
	buff = make([]byte, size)
	for index := range buff {
		if err = binary.Read(r, binary.BigEndian, &buff[index]); err != nil {
			buff = buff[:index]
			return
		}
	}
	return
}
func writeBytes(w io.Writer, buff []byte) (n int64, err error) {
	n = int64(len(buff))
	err = binary.Write(w, binary.BigEndian, buff)
	return
}

func addrWrite(w io.Writer, addr netip.Addr) (n int64, err error) {
	if addr.Is6() {
		if _, err = writeU8(w, 6); err != nil {
			n = 0
			return
		} else if _, err = writeBytes(w, addr.AsSlice()); err != nil {
			n = 1
			return
		}
		n = 17
		return
	}
	if _, err = writeU8(w, 4); err != nil {
		n = 0
		return
	} else if _, err = writeBytes(w, addr.AsSlice()); err != nil {
		n = 1
		return
	}
	n = 5
	return
}
func addrRead(r io.Reader) (addr netip.Addr, n int64, err error) {
	var buff []byte
	n = 1
	switch readU8(r) {
	case 4:
		buff, err = readByteN(r, 4)
		n += int64(len(buff))
		if err != nil {
			return
		}
		addr = netip.AddrFrom4([4]byte(buff))
		return
	case 6:
		buff, err = readByteN(r, 16)
		n += int64(len(buff))
		if err != nil {
			return
		}
		netip.AddrFrom16([16]byte(buff))
		return
	}
	err = fmt.Errorf("connet get ip type")
	return
}

func addrPortRead(r io.Reader) (netip.AddrPort, int64, error) {
	switch readU8(r) {
	case 4:
		buff, err := readByteN(r, 4)
		if err != nil {
			return netip.AddrPort{}, int64(len(buff)), err
		}
		return netip.AddrPortFrom(netip.AddrFrom4([4]byte(buff)), readU16(r)), 6, nil
	case 6:
		buff, err := readByteN(r, 16)
		if err != nil {
			return netip.AddrPort{}, int64(len(buff)), err
		}
		return netip.AddrPortFrom(netip.AddrFrom16([16]byte(buff)), readU16(r)), 19, nil
	}
	return netip.AddrPort{}, 1, fmt.Errorf("connet get ip type")
}
func addrPortWrite(w io.Writer, addr netip.AddrPort) (n int64, err error) {
	if !addr.IsValid() {
		return 0, fmt.Errorf("invalid ip address")
	} else if addr.Addr().Is6() {
		if _, err = writeU8(w, 6); err != nil {
			return 0, err
		} else if err = binary.Write(w, binary.BigEndian, addr.Addr().AsSlice()); err != nil {
			return 1, err
		}
		n = 18
		return
	}
	if _, err = writeU8(w, 4); err != nil {
		return 0, err
	} else if err = binary.Write(w, binary.BigEndian, addr.Addr().AsSlice()); err != nil {
		return 1, err
	}
	n = 5
	return
}

func writeOption(w io.Writer, d any, callback func(w io.Writer) (n int64, err error)) (n int64, err error) {
	if d == nil {
		return writeU8(w, 0)
	}
	n, err = writeU8(w, 1)
	if err != nil {
		return
	}
	n2, err2 := callback(w)
	return n + n2, err2
}
func readOption(r io.Reader, callback func(r io.Reader) (n int64, err error)) (n int64, err error) {
	n = 1
	switch readU8(r) {
	case 0:
		return 1, nil
	case 1:
		n2, err := callback(r)
		return n + n2, err
	}
	return 1, fmt.Errorf("invalid Option value")
}
