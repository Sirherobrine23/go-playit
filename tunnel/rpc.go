package tunnel

import (
	"io"
)

type MessageEncoding interface {
	WriteTo(I io.Writer) error
	ReadFrom(I io.Reader) error
}

type ControlRpcMessage[T MessageEncoding] struct {
	RequestID uint64
	Content   T // Convert with .(*type)
}

func (w *ControlRpcMessage[T]) WriteTo(I io.Writer) error {
	if err := WriteU64(I, w.RequestID); err != nil {
		return err
	}
	return w.Content.WriteTo(I)
}

func (w *ControlRpcMessage[T]) ReadFrom(I io.Reader) error {
	w.RequestID = ReadU64(I)
	return w.Content.ReadFrom(I)
}