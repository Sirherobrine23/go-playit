package claim

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"sirherobrine23.org/playit-cloud/go-playit/api"
	"sirherobrine23.org/playit-cloud/go-playit/internal/request"
)

type Claim struct {
	Code      string `json:"code"`       // Claim code
	Agent     string `json:"agent_type"` // "default" | "assignable" | "self-managed"
	Version   string `json:"version"`    // Project version
	SecretKey string `json:"-"`
}

func MakeClaim() (*Claim, error) {
	cl, buf := Claim{}, make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}

	cl.Code = hex.EncodeToString(buf)
	return &cl, nil
}

func (w *Claim) Setup() error {
	var Version string
	info, ok := debug.ReadBuildInfo()
	if !ok {
		Version = "unknown"
	} else {
		for _, dep := range info.Deps {
			if dep.Path == "sirherobrine23.org/playit-cloud/go-playit" {
				Version = dep.Version
				break
			}
		}
	}
	w.Version = fmt.Sprintf("go-playit %s", Version)
	req := request.RequestOptions{
		Method: "POST",
		Url:    fmt.Sprintf("%s/claim/setup", api.PlayitAPI),
	}
	body, err := json.MarshalIndent(&w, "", "  ")
	if err != nil {
		return err
	}

	for {
		var res struct {
			Data string `json:"data"`
		}
		req.Body = bytes.NewReader(body)
		if _, err := req.Do(&res); err != nil {
			return err
		}

		// WaitingForUserVisit, WaitingForUser, UserAccepted, UserRejected
		if res.Data == "WaitingForUserVisit" || res.Data == "WaitingForUser" {
			time.Sleep(time.Millisecond + 200)
			continue
		} else if res.Data == "UserRejected" {
			return fmt.Errorf("claim rejected")
		} else if res.Data == "UserAccepted" {
			break
		}
	}

	if body, err = json.MarshalIndent(&struct{ code string }{w.Code}, "", "  "); err != nil {
		return err
	}

	var data struct {
		status string
		data   any
	}
	req.Body = bytes.NewReader(body)
	if _, err := req.Do(&data); err != nil {
		return err
	} else if data.status != "success" {
		return fmt.Errorf("cannot get secret key")
	}

	w.SecretKey = data.data.(struct{ secret_key string }).secret_key
	return nil
}
