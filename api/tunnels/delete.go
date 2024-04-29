package tunnels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"sirherobrine23.org/playit-cloud/go-agent/api"
	"sirherobrine23.org/playit-cloud/go-agent/internal/request"
)

func Delete(secret string, tunID string) error {
	body, err := json.MarshalIndent(struct {
		Tunnel string `json:"tunnel_id"`
	}{tunID}, "", "  ")
	if err != nil {
		return err
	}

	req := request.RequestOptions{
		Method: "POST",
		Url:    fmt.Sprintf("%s/tunnels/create", api.PlayitAPI),
		Body:   bytes.NewReader(body),
		Headers: http.Header{
			"x-content-type": {"application/json"},
			"x-accepts":      {"application/json"},
		},
	}
	if secret = strings.TrimSpace(secret); len(secret) > 0 {
		req.Headers.Set("Authorization", fmt.Sprintf("Agent-Key %s", secret))
	}

	var status struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
	}
	res, err := req.Do(&status)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 {
		return nil
	} else if res.StatusCode == 400 {
		info := status.Data.(struct{ message string })
		return fmt.Errorf(info.message)
	} else if res.StatusCode == 401 {
		return fmt.Errorf("invaid secret")
	}
	return fmt.Errorf("backend error, code: %d (%s)", res.StatusCode, res.Status)
}
