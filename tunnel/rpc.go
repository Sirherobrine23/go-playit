package tunnel

import (
	"encoding/binary"
	"io"
)

type MessageEncoding interface {
	io.ReaderFrom
	io.WriterTo
}

type ControlRpcMessage[T MessageEncoding] struct {
	RequestID uint64
	Content   T // Convert with .(*type)
}

func (rpc *ControlRpcMessage[T]) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.BigEndian, rpc.RequestID); err != nil {
		return 0, err
	} else if n, err = rpc.Content.WriteTo(w); err != nil {
		return 8, err
	}
	n += 8
	return
}
func (rpc *ControlRpcMessage[T]) ReadFrom(r io.Reader) (n int64, err error) {
	if err = binary.Read(r, binary.BigEndian, &rpc.RequestID); err != nil {
		n = 0
		return n, err
	} else if n, err = rpc.Content.ReadFrom(r); err != nil {
		return 8, err
	}
	n += 8
	return
}
