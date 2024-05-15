package request

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	Error401 error = fmt.Errorf("check token or authentication")
)

type Request struct {
	Base     string            // https://api.example.com
	Token    string            // Request Token
	AgentKey string            // Agent Key
	Headers  map[string]string // Request headers
}

func (reqBase *Request) Request(Method, PathRequest string, Body io.Reader) (*http.Response, error) {
	if len(Method) == 0 {
		Method = "GET"
	}

	req, err := http.NewRequest(strings.ToUpper(Method), fmt.Sprintf("%s%s", reqBase.Base, PathRequest), Body)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{}
	for key, value := range reqBase.Headers {
		req.Header.Set(key, value)
	}

	if len(reqBase.Token) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", reqBase.Token))
	}
	if len(reqBase.AgentKey) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Agent-Key %s", reqBase.AgentKey))
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode == 401 {
		return res, Error401
	}
	return res, nil
}
