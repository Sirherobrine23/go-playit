package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func recodeJson(from, to any) error {
	buff, err := json.Marshal(&from)
	if err != nil {
		return err
	} else if err = json.Unmarshal(buff, to); err != nil {
		return err
	}
	return nil
}

func requestToApi(Path, Token string, Body io.Reader, Response any, Headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", PlayitAPI, Path), Body)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{}
	for key, value := range Headers {
		req.Header.Set(key, value)
	}

	// Set agent token
	if len(Token) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Agent-Key %s", Token))
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var ResBody struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&ResBody); err != nil {
		return res, err
	}

	if res.StatusCode >= 300 {
		var errStatus struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if data, is := ResBody.Data.(string); is {
			return res, fmt.Errorf(data)
		} else if err = recodeJson(&ResBody.Data, &errStatus); err != nil {
			return res, err
		}
		if len(errStatus.Message) > 0 {
			return res, fmt.Errorf("%s: %s", errStatus.Type, errStatus.Message)
		}
		return res, fmt.Errorf("%s", errStatus.Type)
	}
	if Response != nil {
		if err = recodeJson(&ResBody.Data, Response); err != nil {
			return res, err
		}
	}
	return res, nil
}
