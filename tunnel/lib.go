package tunnel

import (
	"io"
)

type AgentSessionId struct {
	SessionID, AccountID, AgentID uint64
}

func (w *AgentSessionId) WriteTo(I io.Writer) error {
	var err error
	if err = WriteU64(I, w.SessionID); err != nil {
		return err
	} else if err = WriteU64(I, w.AccountID); err != nil {
		return err
	} else if err = WriteU64(I, w.AgentID); err != nil {
		return err
	}

	return nil
}

func (w *AgentSessionId) ReadFrom(I io.Reader) error {
	w.SessionID, w.AccountID, w.AgentID = ReadU64(I), ReadU64(I), ReadU64(I)
	return nil
}