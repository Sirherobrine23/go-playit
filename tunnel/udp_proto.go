package tunnel

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
)

const (
	REDIRECT_FLOW_4_FOOTER_ID_OLD uint64 = 0x5cb867cf788173b2
	REDIRECT_FLOW_4_FOOTER_ID     uint64 = 0x4448474f48414344
	REDIRECT_FLOW_6_FOOTER_ID     uint64 = 0x6668676f68616366
	UDP_CHANNEL_ESTABLISH_ID      uint64 = 0xd01fe6830ddce781

	V4_LEN int = 20
	V6_LEN int = 48
)

type UdpFlow struct {
	IPSrc, IPDst netip.AddrPort
	Flow uint32
}

func (w *UdpFlow) Len() int {
	if w.IPSrc.Addr().Is4() {
		return V4_LEN
	}
	return V6_LEN
}

func (w *UdpFlow) Src() netip.AddrPort {
	return w.IPSrc
}
func (w *UdpFlow) Dst() netip.AddrPort {
	return w.IPDst
}

func (w *UdpFlow) WithSrcPort(port uint16) UdpFlow {
	return UdpFlow{
		IPSrc: netip.AddrPortFrom(w.IPSrc.Addr(), port),
		IPDst: w.IPSrc,
	}
}

func (w *UdpFlow) WriteTo(writer io.Writer) error {
	if err := enc.WriteBytes(writer, w.IPSrc.Addr().AsSlice()); err != nil {
		return err
	} else if err := enc.WriteBytes(writer, w.IPDst.Addr().AsSlice()); err != nil {
		return err
	} else if err := enc.WriteU16(writer, w.IPSrc.Port()); err != nil {
		return err
	} else if err := enc.WriteU16(writer, w.IPDst.Port()); err != nil {
		return err
	}

	if w.IPSrc.Addr().Is6() {
		if err := enc.WriteU32(writer, w.Flow); err != nil {
			return err
		} else if err := enc.WriteU64(writer, REDIRECT_FLOW_6_FOOTER_ID); err != nil {
			return err
		}
	} else {
		if err := enc.WriteU64(writer, REDIRECT_FLOW_4_FOOTER_ID_OLD); err != nil {
			return err
		}
	}

	return nil
}

func FromTailUdpFlow(slice []byte) (UdpFlow, uint64, error) {
	debug.Printf("FromTailUdpFlow: Avaible bytes: %+v\n", slice)
	if len(slice) < 8 {
		return UdpFlow{}, 0, fmt.Errorf("not space to footer")
	}
	footer := binary.BigEndian.Uint64(slice[(len(slice)-8):])
	debug.Printf("FromTailUdpFlow: Footer %d, bytes: %+v\n", footer, slice[(len(slice)-8):])
	if footer == REDIRECT_FLOW_4_FOOTER_ID || footer == REDIRECT_FLOW_4_FOOTER_ID_OLD || footer == (REDIRECT_FLOW_4_FOOTER_ID | REDIRECT_FLOW_4_FOOTER_ID_OLD) {
		if len(slice) < V4_LEN {
			return UdpFlow{}, 0, fmt.Errorf("v4 not have space")
		}
		debug.Printf("FromTailUdpFlow: bytes v4: %+v\n", slice[len(slice)-V4_LEN:])
		reader := bytes.NewReader(slice[len(slice)-V4_LEN:])

		var err error
		var src_ip, dst_ip []byte
		if src_ip, err = enc.ReadByteN(reader, 4); err != nil {
			return UdpFlow{}, 0, err
		} else if dst_ip, err = enc.ReadByteN(reader, 4); err != nil {
			return UdpFlow{}, 0, err
		}
		src_port, dst_port := enc.ReadU16(reader), enc.ReadU16(reader)
		srcIP := netip.AddrFrom4([4]byte(src_ip))
		dstIP := netip.AddrFrom4([4]byte(dst_ip))

		var point UdpFlow
		point.IPSrc = netip.AddrPortFrom(srcIP, src_port)
		point.IPDst = netip.AddrPortFrom(dstIP, dst_port)
		return point, 0, nil
	} else if footer == REDIRECT_FLOW_6_FOOTER_ID {
		if len(slice) < V6_LEN {
			return UdpFlow{}, footer, fmt.Errorf("v6 not have space")
		}
		debug.Printf("FromTailUdpFlow: bytes v4: %+v\n", slice[len(slice)-V6_LEN:])
		reader := bytes.NewReader(slice[len(slice)-V6_LEN:])

		var err error
		var src_ip, dst_ip []byte
		if src_ip, err = enc.ReadByteN(reader, 16); err != nil {
			return UdpFlow{}, 0, err
		} else if dst_ip, err = enc.ReadByteN(reader, 16); err != nil {
			return UdpFlow{}, 0, err
		}
		src_port, dst_port, flow := enc.ReadU16(reader), enc.ReadU16(reader), enc.ReadU32(reader)
		srcIP := netip.AddrFrom16([16]byte(src_ip))
		dstIP := netip.AddrFrom16([16]byte(dst_ip))

		var point UdpFlow
		point.IPSrc = netip.AddrPortFrom(srcIP, src_port)
		point.IPDst = netip.AddrPortFrom(dstIP, dst_port)
		point.Flow = flow
		return point, 0, nil
	}
	debug.Printf("Cannot reader tail udp flow, bytes: %+v\n", slice)
	return UdpFlow{}, footer, fmt.Errorf("read fotter")
}
