package messages

import (
	"encoding/binary"
)

type RequestBodyWriter struct {
	buf []byte
}

func RequestId(buf []byte, id int64) *RequestBodyWriter {
	binary.BigEndian.PutUint64(buf, uint64(id))
	return &RequestBodyWriter{buf: buf}
}

func (w *RequestBodyWriter) Ping(now int64, sessionId *AgentSessionId) { // Assuming AgentSessionId exists
	binary.BigEndian.PutUint32(w.buf, 1) // Ping ID
	w.buf = w.buf[4:]                    // Advance buffer position

	binary.BigEndian.PutUint64(w.buf, uint64(now))
	w.buf = w.buf[8:]

	if sessionId == nil {
		w.buf = append(w.buf, 0) // Null marker
	} else {
		w.buf = append(w.buf, 1) // Session exists marker
		sessionId.writeTo(w.buf) // Write session data
	}
}

func (w *RequestBodyWriter) KeepAlive(sessionId *AgentSessionId) {
	w.buf = append(w.buf, 3)
	sessionId.writeTo(w.buf)
}

func (w *RequestBodyWriter) RegisterBytes(signedRegisterBytes []byte) {
	w.buf = append(w.buf, signedRegisterBytes...)
}