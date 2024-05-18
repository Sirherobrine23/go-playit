package tunnel

import (
	"fmt"
	"io"
)

type MessageEncoding interface {
	WriteTo(I io.Writer) error
	ReadFrom(I io.Reader) error
}

type ControlRpcMessage struct {
	RequestID uint64
	Content   any
}

func (w *ControlRpcMessage) WriteTo(I io.Writer) error {
	contentWrite, isEncoding := w.Content.(MessageEncoding)
	if !isEncoding {
		return fmt.Errorf("Content not is MessageEncoding")
	} else if err := WriteU64(I, w.RequestID); err != nil {
		return err
	}
	return contentWrite.WriteTo(I)
}

func (w *ControlRpcMessage) ReadFrom(I io.Reader) error {
	contentWrite, isEncoding := w.Content.(MessageEncoding)
	if !isEncoding {
		return fmt.Errorf("Content not is MessageEncoding")
	}
	w.RequestID = ReadU64(I)
	return contentWrite.ReadFrom(I)
}