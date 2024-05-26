package proto

import (
	"io"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
)

type ControlRpcMessage[T MessageEncoding] struct {
	RequestID uint64
	Content   T // Convert with .(*type)
}

func (rpc *ControlRpcMessage[T]) WriteTo(w io.Writer) error {
	if err := enc.WriteU64(w, rpc.RequestID); err != nil {
		return err
	} else if err = rpc.Content.WriteTo(w); err != nil {
		return err
	}
	return nil
}
func (rpc *ControlRpcMessage[T]) ReadFrom(r io.Reader) error {
	rpc.RequestID = enc.ReadU64(r)
	if err := rpc.Content.ReadFrom(r); err != nil {
		return err
	}
	return nil
}
