package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func recodeJson(from, to any) error {
	buff, err := json.MarshalIndent(&from, "", "  ")
	if err != nil {
		return err
	} else if err = json.Unmarshal(buff, to); err != nil {
		return err
	}
	return nil
}

func prettyJSON(from string) string {
	var data any
	if err := json.Unmarshal([]byte(from), &data); err != nil {
		return from
	}
	marData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return from
	}
	return string(marData)
}

func (w *Api) requestToApi(Path string, Body io.Reader, Response any, Headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", PlayitAPI, Path), Body)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{}
	for key, value := range Headers {
		req.Header.Set(key, value)
	}

	// Set agent token
	if len(w.Secret) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Agent-Key %s", w.Secret))
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return res, err
	}
	debug.Printf("(%q %d): %s\n", Path, res.StatusCode, prettyJSON(string(data)))

	var ResBody struct {
		Error  string `json:"error"`
		Status string `json:"status"`
		Data   any    `json:"data"`
	}
	if err = json.Unmarshal(data, &ResBody); err != nil {
		return res, err
	}
	if res.StatusCode >= 300 {
		if ResBody.Error != "" {
			return res, fmt.Errorf("api.playit.gg: %s", ResBody.Error)
		}
		var errStatus struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if data, is := ResBody.Data.(string); is {
			return res, fmt.Errorf("api.playit.gg: %s", data)
		} else if err = recodeJson(&ResBody.Data, &errStatus); err != nil {
			return res, err
		}
		if len(errStatus.Message) > 0 {
			return res, fmt.Errorf("api.playit.gg: %s %s", errStatus.Type, errStatus.Message)
		}
		return res, fmt.Errorf("api.playit.gg: %s", errStatus.Type)
	}
	if Response != nil {
		if err = recodeJson(&ResBody.Data, Response); err != nil {
			return res, err
		}
	}
	return res, nil
}
