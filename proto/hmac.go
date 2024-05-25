package proto

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
)

type HmacSha256 struct {
	mac hash.Hash
}

func NewHmacSha256(secret []byte) *HmacSha256 {
	mac := hmac.New(sha256.New, secret)
	return &HmacSha256{mac}
}

func (h *HmacSha256) Verify(data, sig []byte) bool {
	expectedMAC := h.mac.Sum(nil)
	h.mac.Reset()
	h.mac.Write(data)
	return hmac.Equal(expectedMAC, sig)
}

func (h *HmacSha256) Sign(data []byte) [32]byte {
	h.mac.Reset()
	h.mac.Write(data)
	return [32]byte(h.mac.Sum(nil))
}
