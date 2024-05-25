package proto

import (
	"fmt"
	"io"
)

type RawSlice []byte

func (buff RawSlice) ReadFrom(r io.Reader) error {
	return fmt.Errorf("cannot read for RawSlice")
}
func (buff RawSlice) WriteTo(w io.Writer) error {
	size, err := w.Write(buff)
	if err != nil {
		return err
	} else if size != len(buff) {
		return fmt.Errorf("not enough space to write raw slice")
	}
	return nil
}
