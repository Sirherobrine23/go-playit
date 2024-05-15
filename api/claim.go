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

	"sirherobrine23.org/playit-cloud/go-playit/internal/request"
)

type Claim struct {
	Code      string `json:"code"`       // Claim code
	Agent     string `json:"agent_type"` // "default" | "assignable" | "self-managed"
	Version   string `json:"version"`    // Project version
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

	req := request.Request{
		Base:    PlayitAPI,
		Headers: map[string]string{},
	}

	var assignSecretRequestBody []byte
	var err error
	if assignSecretRequestBody, err = json.MarshalIndent(&w, "", "  "); err != nil {
		return "", err
	}

	for {
		res, err := req.Request("POST", "/claim/setup", bytes.NewReader(assignSecretRequestBody))
		if err != nil {
			return "", err
		}

		var waitUser struct {
			Data string `json:"data"`
		}
		err = json.NewDecoder(res.Body).Decode(&waitUser)
		res.Body.Close()
		if err != nil {
			return "", err
		}

		// WaitingForUserVisit, WaitingForUser, UserAccepted, UserRejected
		if waitUser.Data == "WaitingForUserVisit" || waitUser.Data == "WaitingForUser" {
			time.Sleep(time.Millisecond + 200)
			continue
		} else if waitUser.Data == "UserRejected" {
			return "", fmt.Errorf("claim rejected")
		} else if waitUser.Data == "UserAccepted" {
			break
		}
	}

	exchangeBody, err := json.Marshal(&struct {Code string `json:"code"`}{w.Code})
	if err != nil {
		return "", err
	}

	res, err := req.Request("POST", "/claim/exchange", bytes.NewBuffer(exchangeBody));
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	var requestSecret struct {
		Data struct {
			SecretKey string `json:"secret_key"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&requestSecret); err != nil {
		return "", err
	}

	// Set secret to base
	return requestSecret.Data.SecretKey, nil
}
