package tunnel

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type ClaimInstructions struct {
	Address AddressPort
	Token   []byte
}

func (w *ClaimInstructions) WriteTo(I io.Writer) error {
	if err := w.Address.WriteTo(I); err != nil {
		return err
	} else if err := WriteU64(I, uint64(len(w.Token))); err != nil {
		return err
	} else if err = binary.Write(I, binary.BigEndian, w.Token); err != nil {
		return err
	}
	return nil
}
func (w *ClaimInstructions) ReadFrom(I io.Reader) error {
	w.Address = AddressPort{}
	if err := w.Address.ReadFrom(I); err != nil {
		return err
	}
	w.Token = make([]byte, ReadU64(I))
	if err := ReadBuff(I, w.Token); err != nil {
		return err
	}
	return nil
}

type NewClient struct {
	ConnectAddr       AddressPort
	PeerAddr          AddressPort
	ClaimInstructions ClaimInstructions
	TunnelServerId    uint64
	DataCenterId      uint32
}

func (w *NewClient) WriteTo(I io.Writer) error {
	if err := w.ConnectAddr.WriteTo(I); err != nil {
		return err
	} else if w.PeerAddr.WriteTo(I); err != nil {
		return err
	} else if w.ClaimInstructions.WriteTo(I); err != nil {
		return err
	} else if err := WriteU64(I, w.TunnelServerId); err != nil {
		return err
	} else if err := WriteU32(I, w.DataCenterId); err != nil {
		return err
	}
	return nil
}
func (w *NewClient) ReadFrom(I io.Reader) error {
	w.ConnectAddr, w.PeerAddr = AddressPort{}, AddressPort{}
	if err := w.ConnectAddr.ReadFrom(I); err != nil {
		return err
	} else if err := w.PeerAddr.ReadFrom(I); err != nil {
		return err
	} else if err := w.ClaimInstructions.ReadFrom(I); err != nil {
		return err
	}
	w.TunnelServerId, w.DataCenterId = ReadU64(I), ReadU32(I)
	return nil
}

type ControlFeed struct {
	Response  *ControlRpcMessage[*ControlResponse]
	NewClient *NewClient
}

func (w *ControlFeed) WriteTo(I io.Writer) error {
	defer func(){
		d, _ := json.MarshalIndent(w, "", "  ")
		LogDebug.Printf("Write Feed: %s\n", string(d))
	}()
	if w.Response != nil {
		if err := WriteU32(I, 1); err != nil {
			return err
		}
		return w.Response.WriteTo(I)
	} else if w.NewClient != nil {
		if err := WriteU32(I, 2); err != nil {
			return err
		}
		return w.NewClient.WriteTo(I)
	}
	return fmt.Errorf("set ResponseControl or NewClient")
}
func (w *ControlFeed) ReadFrom(I io.Reader) error {
	defer func(){
		d, _ := json.MarshalIndent(w, "", "  ")
		LogDebug.Printf("Read Feed: %s\n", string(d))
	}()
	switch ReadU32(I) {
	case 1:
		w.Response = &ControlRpcMessage[*ControlResponse]{}
		w.Response.Content = &ControlResponse{}
		return w.Response.ReadFrom(I)
	case 2:
		w.NewClient = &NewClient{}
		return w.NewClient.ReadFrom(I)
	}
	return fmt.Errorf("invalid ControlFeed id")
}
