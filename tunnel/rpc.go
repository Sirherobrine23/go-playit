package tunnel

import (
	"encoding/json"
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
	defer func() {
		d, _ := json.MarshalIndent(w, "", "  ")
		LogDebug.Printf("Write RPC: %s\n", string(d))
	}()
	return w.Content.WriteTo(I)
}

func (w *ControlRpcMessage[T]) ReadFrom(I io.Reader) error {
	w.RequestID = ReadU64(I)
	defer func() {
		d, _ := json.MarshalIndent(w, "", "  ")
		LogDebug.Printf("Read RPC: %s\n", string(d))
	}()
	return w.Content.ReadFrom(I)
}
