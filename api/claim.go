package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"time"
)

var (
	ClaimAgents []string = []string{
		"default", "assignable", "self-managed",
	}
)

func (w *Api) AiisgnClaimCode() (err error) {
	if len(w.Code) > 0 {
		return nil
	}

	// Make code buffer
	codeBuff := make([]byte, 5)
	if _, err = rand.Read(codeBuff); err != nil {
		return err
	}

	// Convert to hex string
	w.Code = hex.EncodeToString(codeBuff)
	return nil
}

// Get claim url
func (w *Api) ClaimUrl() string {
	return fmt.Sprintf("https://playit.gg/claim/%s", url.PathEscape(w.Code))
}

func (w *Api) ClaimAgentSecret(AgentType string) error {
	if w.Code == "" {
		return fmt.Errorf("assign claim code")
	} else if w.Secret != "" {
		return fmt.Errorf("agent secret key ared located")
	} else if !slices.Contains(ClaimAgents, AgentType) {
		return fmt.Errorf("set valid agent type")
	}

	type Claim struct {
		Code    string `json:"code"`       // Claim code
		Agent   string `json:"agent_type"` // "default" | "assignable" | "self-managed"
		Version string `json:"version"`    // Project version
	}
	type Code struct {
		Code      string `json:"code,omitempty"`
		SecretKey string `json:"secret_key,omitempty"`
	}

	assignSecretRequestBody, err := json.Marshal(Claim{
		Code:    w.Code,
		Agent:   AgentType,
		Version: fmt.Sprintf("go-playit %s", GoPlayitVersion),
	})
	if err != nil {
		return err
	}

	for {
		var waitUser string
		_, err = w.requestToApi("/claim/setup", bytes.NewReader(assignSecretRequestBody[:]), &waitUser, nil)
		if err != nil {
			return err
		}

		if waitUser == "UserRejected" {
			return fmt.Errorf("claim rejected")
		} else if waitUser == "UserAccepted" {
			break
		}
		// wait for request
		time.Sleep(time.Second)
	}
	var getCode Code
	getCode.Code = w.Code

	// Code to json
	exchangeBody, err := json.Marshal(getCode)
	if err != nil {
		return err
	}

	_, err = w.requestToApi("/claim/exchange", bytes.NewBuffer(exchangeBody), &getCode, nil)
	if err != nil {
		return err
	}
	w.Secret = getCode.SecretKey // Copy secret key to Api struct

	return nil
}
