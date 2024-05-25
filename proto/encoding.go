package proto

import (
	"io"
)

type MessageEncoding interface {
	ReadFrom(r io.Reader) error
	WriteTo(w io.Writer) error
}
