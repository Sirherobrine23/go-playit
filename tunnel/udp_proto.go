package tunnel

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

const (
	REDIRECT_FLOW_4_FOOTER_ID_OLD uint64 = 0x5cb867cf788173b2
	REDIRECT_FLOW_4_FOOTER_ID     uint64 = 0x4448474f48414344
	REDIRECT_FLOW_6_FOOTER_ID     uint64 = 0x6668676f68616366
	UDP_CHANNEL_ESTABLISH_ID      uint64 = 0xd01fe6830ddce781

	V4_LEN int = 20
	V6_LEN int = 48
)

type UdpFlowBase struct {
	Src, Dst netip.AddrPort
}

type UdpFlow struct {
	V4 *UdpFlowBase
	V6 *struct {
		UdpFlowBase
		Flow uint32
	}
}

func (w *UdpFlow) Len() int {
	if w.V4 == nil {
		return V6_LEN
	}
	return V4_LEN
}

func (w *UdpFlow) Src() netip.AddrPort {
	if w.V4 == nil {
		return w.V6.UdpFlowBase.Src
	}
	return w.V4.Src
}
func (w *UdpFlow) Dst() netip.AddrPort {
	if w.V4 == nil {
		return w.V6.UdpFlowBase.Dst
	}
	return w.V4.Dst
}

func (w *UdpFlow) WriteTo(writer io.Writer) error {
	var conn UdpFlowBase
	if w.V4 != nil {
		conn = *w.V4
	} else {
		conn = w.V6.UdpFlowBase
	}
	if err := WriteData(writer, conn.Src.Addr().AsSlice()); err != nil {
		return err
	} else if err := WriteData(writer, conn.Dst.Addr().AsSlice()); err != nil {
		return err
	} else if err := WriteU16(writer, conn.Src.Port()); err != nil {
		return err
	} else if err := WriteU16(writer, conn.Dst.Port()); err != nil {
		return err
	}

	if w.V4 != nil {
		if err := WriteU64(writer, REDIRECT_FLOW_4_FOOTER_ID_OLD); err != nil {
			return err
		}
	} else {
		if err := WriteU32(writer, w.V6.Flow); err != nil {
			return err
		} else if err := WriteU64(writer, REDIRECT_FLOW_6_FOOTER_ID); err != nil {
			return err
		}
	}

	return nil
}

func FromTailUdpFlow(slice []byte) (*UdpFlow, uint64, error) {
	if len(slice) < 8 {
		return nil, 0, fmt.Errorf("not space to footer")
	}
	footer := binary.BigEndian.Uint64(slice[len(slice)-8:])
	switch footer {
	case REDIRECT_FLOW_4_FOOTER_ID | REDIRECT_FLOW_4_FOOTER_ID_OLD:
		if len(slice) < V4_LEN {
			return nil, 0, fmt.Errorf("v4 not have space")
		}
		slice = slice[len(slice)-V4_LEN:]
		src_ip, _ := ReadBuffN(bytes.NewReader(slice), 4)
		srcIP, _ := netip.AddrFromSlice(src_ip)
		dst_ip, _ := ReadBuffN(bytes.NewReader(slice), 4)
		dstIP, _ := netip.AddrFromSlice(dst_ip)
		src_port, dst_port := ReadU16(bytes.NewReader(slice)), ReadU16(bytes.NewReader(slice))

		return &UdpFlow{
			V4: &UdpFlowBase{
				Src: netip.AddrPortFrom(srcIP, src_port),
				Dst: netip.AddrPortFrom(dstIP, dst_port),
			},
		}, 0, nil
	case REDIRECT_FLOW_6_FOOTER_ID:
		if len(slice) < V6_LEN {
			return nil, footer, fmt.Errorf("v6 not have space")
		}
		slice = slice[len(slice)-V6_LEN:]
		src_ip, _ := ReadBuffN(bytes.NewReader(slice), 16)
		srcIP, _ := netip.AddrFromSlice(src_ip)
		dst_ip, _ := ReadBuffN(bytes.NewReader(slice), 16)
		dstIP, _ := netip.AddrFromSlice(dst_ip)
		src_port, dst_port := ReadU16(bytes.NewReader(slice)), ReadU16(bytes.NewReader(slice))
		flow := ReadU32(bytes.NewReader(slice))

		return &UdpFlow{
			V6: &struct {
				UdpFlowBase
				Flow uint32
			}{
				UdpFlowBase{
					Src: netip.AddrPortFrom(srcIP, src_port),
					Dst: netip.AddrPortFrom(dstIP, dst_port),
				},
				flow,
			},
		}, 0, nil
	}
	return nil, footer, nil
}
