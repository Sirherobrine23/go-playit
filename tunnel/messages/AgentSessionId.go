package messages

import (
	"encoding/binary"
)

type AgentSessionId  struct {
	sessionId int64
	accountId int64
	agentId   int64
}

func (s *AgentSessionId ) writeTo(buf []byte) {
	binary.BigEndian.PutUint64(buf, uint64(s.sessionId))
	binary.BigEndian.PutUint64(buf[8:], uint64(s.accountId))
	binary.BigEndian.PutUint64(buf[16:], uint64(s.agentId))
}

func (s *AgentSessionId ) readFrom(buf []byte) {
	s.sessionId = int64(binary.BigEndian.Uint64(buf))
	s.accountId = int64(binary.BigEndian.Uint64(buf[8:]))
	s.agentId = int64(binary.BigEndian.Uint64(buf[16:]))
}