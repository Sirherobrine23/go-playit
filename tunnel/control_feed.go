package tunnel

import (
	"fmt"
	"io"
	"net/netip"
)

var (
	ErrFeedRead error = fmt.Errorf("invalid controlFeed id")
)

type ControlFeed struct {
	Response  *ControlRpcMessage[*ControlResponse]
	NewClient *NewClient
}

func (Feed *ControlFeed) ReadFrom(r io.Reader) (n int64, err error) {
	id := readU32(r)
	if id == 1 {
		Feed.Response = new(ControlRpcMessage[*ControlResponse])
		Feed.Response.Content = new(ControlResponse)
		n, err = Feed.Response.ReadFrom(r)
		n += 4
		return
	} else if id == 2 {
		Feed.NewClient = &NewClient{}
		n, err = Feed.NewClient.ReadFrom(r)
		n += 4
		return
	}
	return 4, ErrFeedRead
}
func (Feed *ControlFeed) WriteTo(w io.Writer) (n int64, err error) {
	if Feed.Response != nil {
		if err := writeU32(w, 1); err != nil {
			return 0, err
		}
		n, err = Feed.Response.WriteTo(w)
		n += 4
		return
	} else if Feed.NewClient != nil {
		if err := writeU32(w, 2); err != nil {
			return 0, err
		}
		n, err = Feed.NewClient.WriteTo(w)
		n += 4
		return
	}
	return 0, fmt.Errorf("")
}

type NewClient struct {
	ConnectAddr       netip.AddrPort
	PeerAddr          netip.AddrPort
	ClaimInstructions ClaimInstructions
	TunnelServerId    uint64
	DataCenterId      uint32
}

func (client *NewClient) ReadFrom(r io.Reader) (n int64, err error) {
	client.ConnectAddr, n, err = addrPortRead(r)
	if err != nil {
		return n, err
	}

	n2 := n
	client.PeerAddr, n, err = addrPortRead(r)
	if err != nil {
		return n2 + n, err
	}

	n3 := n2 + n
	n, err = client.ClaimInstructions.ReadFrom(r);
	if err != nil {
		return n3 + n, err
	}
	n+=n3 + 8 + 4
	client.TunnelServerId, client.DataCenterId = readU64(r), readU32(r)
	return
}
func (client *NewClient) WriteTo(w io.Writer) (n int64, err error) {
	n, err = addrPortWrite(w, client.ConnectAddr)
	if err != nil {
		return n, err
	}
	n2 := n
	n, err = addrPortWrite(w, client.PeerAddr)
	if err != nil {
		return n+n2, err
	}
	n3:= n+n2
	if n, err = client.ClaimInstructions.WriteTo(w); err != nil {
		return n + n3, err
	}

	n4 := n + n3
	if err = writeU64(w, client.TunnelServerId); err != nil {
		return n4, err
	}
	n4 += 8
	if err = writeU32(w, client.DataCenterId); err != nil {
		return n4, err
	}
	n = n4+8
	return
}

type ClaimInstructions struct {
	Address netip.AddrPort
	Token   []byte
}

func (claim *ClaimInstructions) ReadFrom(r io.Reader) (n int64, err error) {
	claim.Address, n, err = addrPortRead(r)
	if err != nil {
		return n, err
	}
	claim.Token, err = readByteN(r, int(readU64(r)))
	n += int64(len(claim.Token)) + 8
	return
}
func (claim *ClaimInstructions) WriteTo(w io.Writer) (n int64, err error) {
	n, err = addrPortWrite(w, claim.Address)
	if err != nil {
		return n, err
	}

	if err = writeU64(w, uint64(len(claim.Token))); err != nil {
		return n, err
	}

	n2 := 8 + n
	n, err = writeBytes(w, claim.Token)
	if err != nil {
		return n2, err
	}
	n = n2 + n
	return
}