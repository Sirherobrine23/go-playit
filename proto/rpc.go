package proto

import (
	"encoding/binary"
	"io"
)

type ControlRpcMessage[T MessageEncoding] struct {
	RequestID uint64
	Content   T // Convert with .(*type)
}

func (rpc *ControlRpcMessage[T]) WriteTo(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, rpc.RequestID); err != nil {
		return err
	} else if err = rpc.Content.WriteTo(w); err != nil {
		return err
	}
	return nil
}
func (rpc *ControlRpcMessage[T]) ReadFrom(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &rpc.RequestID); err != nil {
		return err
	} else if err = rpc.Content.ReadFrom(r); err != nil {
		return err
	}
	return nil
}
