package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime/debug"
	"time"
)

type Claim struct {
	Secret  string `json:"-"`          // Secret
	Code    string `json:"code"`       // Claim code
	Agent   string `json:"agent_type"` // "default" | "assignable" | "self-managed"
	Version string `json:"version"`    // Project version
}

func MakeClaim(Agent string) (*Claim, error) {
	cl, buf := Claim{}, make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}

	cl.Code = hex.EncodeToString(buf)
	cl.Agent = Agent
	return &cl, nil
}

// Get claim url
func (w *Claim) Url() string {
	return fmt.Sprintf("https://playit.gg/claim/%s", url.PathEscape(w.Code))
}

// Get agent secret key
func (w *Claim) Setup() (string, error) {
	if len(w.Agent) == 0 {
		return "", fmt.Errorf("set agent type")
	}

	// Get agent version
	w.Version = ""
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			if dep.Path == "sirherobrine23.org/playit-cloud/go-playit" {
				w.Version = fmt.Sprintf("go-playit %s", dep.Version)
				break
			}
		}
	}
	if len(w.Version) == 0 {
		return "", fmt.Errorf("cannot get go-playit version")
	}

	var assignSecretRequestBody []byte
	var err error
	if assignSecretRequestBody, err = json.MarshalIndent(&w, "", "  "); err != nil {
		return "", err
	}

	for {
		var waitUser string
		_, err = requestToApi("/claim/setup", "", bytes.NewReader(assignSecretRequestBody), &waitUser, nil)
		if err != nil {
			return "", err
		}

		// WaitingForUserVisit, WaitingForUser, UserAccepted, UserRejected
		if waitUser == "WaitingForUserVisit" || waitUser == "WaitingForUser" {
			time.Sleep(time.Millisecond + 200)
			continue
		} else if waitUser == "UserRejected" {
			return "", fmt.Errorf("claim rejected")
		} else if waitUser == "UserAccepted" {
			break
		}
	}

	exchangeBody, err := json.Marshal(&struct {
		Code string `json:"code"`
	}{w.Code})
	if err != nil {
		return "", err
	}

	var requestSecret struct {
		SecretKey string `json:"secret_key"`
	}
	_, err = requestToApi("/claim/exchange", "", bytes.NewBuffer(exchangeBody), &requestSecret, nil)
	if err != nil {
		return "", err
	}

	// Set secret to base
	w.Secret = requestSecret.SecretKey
	return w.Secret, nil
}
