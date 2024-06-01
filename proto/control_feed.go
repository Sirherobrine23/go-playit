package proto

import (
	"fmt"
	"io"
	"net/netip"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

var (
	ErrFeedRead error = fmt.Errorf("invalid controlFeed id")
)

type ControlFeed struct {
	Response  *ControlRpcMessage[*ControlResponse]
	NewClient *NewClient
}

func (Feed *ControlFeed) ReadFrom(r io.Reader) (err error) {
	id := enc.ReadU32(r)
	if id == 1 {
		Feed.Response = new(ControlRpcMessage[*ControlResponse])
		Feed.Response.Content = new(ControlResponse)
		err = Feed.Response.ReadFrom(r)
		debug.Printf("Read Feed (id %d): %s\n", id, logfile.JSONString(Feed))
	} else if id == 2 {
		Feed.NewClient = &NewClient{}
		err = Feed.NewClient.ReadFrom(r)
		debug.Printf("Read Feed (id %d): %s\n", id, logfile.JSONString(Feed))
	} else {
		err = ErrFeedRead
	}
	return
}
func (Feed *ControlFeed) WriteTo(w io.Writer) error {
	defer debug.Printf("Write Feed: %s\n", logfile.JSONString(Feed))
	if Feed.Response != nil {
		if err := enc.WriteU32(w, 1); err != nil {
			return err
		}
		return Feed.Response.WriteTo(w)
	} else if Feed.NewClient != nil {
		if err := enc.WriteU32(w, 2); err != nil {
			return err
		}
		return Feed.NewClient.WriteTo(w)
	}
	return fmt.Errorf("set Response or NewClient")
}

type NewClient struct {
	ConnectAddr       netip.AddrPort
	PeerAddr          netip.AddrPort
	ClaimInstructions ClaimInstructions
	TunnelServerId    uint64
	DataCenterId      uint32
}

func (client *NewClient) ReadFrom(r io.Reader) error {
	var err error
	if client.ConnectAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if client.PeerAddr, err = enc.AddrPortRead(r); err != nil {
		return err
	} else if err = client.ClaimInstructions.ReadFrom(r); err != nil {
		return err
	}
	client.TunnelServerId, client.DataCenterId = enc.ReadU64(r), enc.ReadU32(r)
	return nil
}
func (client *NewClient) WriteTo(w io.Writer) error {
	if err := enc.AddrPortWrite(w, client.ConnectAddr); err != nil {
		return err
	} else if err := enc.AddrPortWrite(w, client.PeerAddr); err != nil {
		return err
	} else if err := client.ClaimInstructions.WriteTo(w); err != nil {
		return err
	} else if err := enc.WriteU64(w, client.TunnelServerId); err != nil {
		return err
	} else if err := enc.WriteU32(w, client.DataCenterId); err != nil {
		return err
	}
	return nil
}

type ClaimInstructions struct {
	Address netip.AddrPort
	Token   []byte
}

func (claim *ClaimInstructions) ReadFrom(r io.Reader) (err error) {
	if claim.Address, err = enc.AddrPortRead(r); err != nil {
		return err
	}
	claim.Token, err = enc.ReadByteN(r, int(enc.ReadU64(r)))
	return
}
func (claim *ClaimInstructions) WriteTo(w io.Writer) error {
	if err := enc.AddrPortWrite(w, claim.Address); err != nil {
		return err
	} else if err := enc.WriteU64(w, uint64(len(claim.Token))); err != nil {
		return err
	} else if err = enc.WriteBytes(w, claim.Token); err != nil {
		return err
	}
	return nil
}
