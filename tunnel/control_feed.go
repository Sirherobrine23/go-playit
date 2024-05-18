package tunnel

import (
	"fmt"
	"io"
)

type ClaimInstructions struct {
	Address AddressPort
	Token   []byte
}

type NewClient struct {
	connect_addr       AddressPort
	peer_addr          AddressPort
	claim_instructions ClaimInstructions
	tunnel_server_id   uint64
	data_center_id     uint32
}
func (w *NewClient) WriteTo(I io.Writer) error {return nil}
func (w *NewClient) ReadFrom(I io.Reader) error {return nil}

type ControlFeed struct {
	Response  *ControlRpcMessage
	NewClient *NewClient
}

func (w *ControlFeed) WriteTo(I io.Writer) error {
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
	return nil
}
func (w *ControlFeed) ReadFrom(I io.Reader) error {
	switch ReadU32(I) {
	case 1:
		w.Response = &ControlRpcMessage{}
		w.Response.Content = &ControlResponse{}
		return w.Response.ReadFrom(I)
	case 2:
		w.NewClient = &NewClient{}
		return w.NewClient.ReadFrom(I)
	}
	return fmt.Errorf("invalid ControlFeed id")
}
