package proto

import (
	"io"
	"reflect"

	"sirherobrine23.org/playit-cloud/go-playit/enc"
	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

type ControlRpcMessage[T MessageEncoding] struct {
	RequestID uint64
	Content   T // Convert with .(*type)
}

func (rpc *ControlRpcMessage[T]) WriteTo(w io.Writer) error {

	defer debug.Printf("Write ControlRpcMessage[%s]: %s\n", reflect.TypeOf(rpc.Content).String(), logfile.JSONString(rpc))
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
		debug.Printf("Read ControlRpcMessage[%s] error: %s\n", reflect.TypeOf(rpc.Content).String(), err.Error())
		return err
	}
	debug.Printf("Read ControlRpcMessage[%s]: %s\n", reflect.TypeOf(rpc.Content).String(), logfile.JSONString(rpc))
	return nil
}
