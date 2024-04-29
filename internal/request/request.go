package request

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

var (
	ErrNoUrl        = errors.New("no url informed")
	ErrPageNotExist = errors.New("page not exists")
)

var DefaultHeader = http.Header{
	"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
	// "Accept-Encoding":           {"gzip, deflate"},
	"Accept-Language":           {"en-US;q=0.9,en;q=0.8"},
	"Sec-Ch-Ua":                 {`"Google Chrome";v="123", "Not:A-Brand";v="8", "Chromium";v="123\"`},
	"Sec-Ch-Ua-Mobile":          {"?0"},
	"Sec-Ch-Ua-Platform":        {`"Windows"`},
	"Sec-Fetch-Dest":            {"document"},
	"Sec-Fetch-Mode":            {"navigate"},
	"Sec-Fetch-Site":            {"none"},
	"Sec-Fetch-User":            {"?1"},
	"Upgrade-Insecure-Requests": {"1"},
	"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"},
}

func arrayHas(arr []int, is int) bool {
	for _, k := range arr {
		if k == is {
			return true
		}
	}
	return false
}

type RequestOptions struct {
	Url         string            // Request url
	HttpError   bool              // Return if status code is equal or then 300
	Method      string            // Request Method
	Body        io.Reader         // Body Reader
	Headers     http.Header       // Extra HTTP Headers
	Querys      map[string]string // Default querys to set in url
	CodesRetrys []int             // Code to retrys in GET method
}

func (w *RequestOptions) ToUrl() (*url.URL, error) {
	if len(w.Url) == 0 {
		return nil, ErrNoUrl
	}

	urlParsed, err := url.Parse(w.Url)
	if err != nil {
		return nil, err
	}

	if len(w.Querys) > 0 {
		query := urlParsed.Query()
		for key, value := range w.Querys {
			query.Set(key, value)
		}
		urlParsed.RawQuery = query.Encode()
	}

	return urlParsed, nil
}

func (w *RequestOptions) String() (string, error) {
	s, err := w.ToUrl()
	if err != nil {
		return "", err
	}
	return s.String(), nil
}

func (opt *RequestOptions) Request() (http.Response, error) {
	if len(opt.Method) == 0 {
		opt.Method = "GET"
	}

	urlRequest, err := opt.ToUrl()
	if err != nil {
		return http.Response{}, err
	}

	// Create request
	req, err := http.NewRequest(opt.Method, urlRequest.String(), opt.Body)
	if err != nil {
		return http.Response{}, err
	}

	// Project headers
	for key, value := range DefaultHeader {
		req.Header[key] = value
	}

	// Set headers
	for key, value := range opt.Headers {
		req.Header[key] = value
	}

	// Create response from request
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return *res, err
	}

	if opt.HttpError && res.StatusCode >= 300 {
		if opt.Method == "GET" && arrayHas(opt.CodesRetrys, res.StatusCode) {
			return opt.Request()
		} else if res.StatusCode == 404 {
			err = ErrPageNotExist
		} else {
			err = fmt.Errorf("response non < 299, code %d, url: %q", res.StatusCode, opt.Url)
		}
	}

	// User tratement
	return *res, err
}

func (opt *RequestOptions) Do(jsonInterface any) (http.Response, error) {
	res, err := opt.Request()
	if err != nil {
		return res, err
	}
	defer res.Body.Close()
	return res, json.NewDecoder(res.Body).Decode(jsonInterface)
}

func (opt *RequestOptions) WriteStream(writer io.Writer) error {
	res, err := opt.Request()
	if err != nil {
		return err
	}

	defer res.Body.Close()
	_, err = io.Copy(writer, res.Body)
	if err != nil {
		return err
	}

	return nil
}

func (opt *RequestOptions) File(filePath string) error {
	// Create file
	localFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Create Request
	res, err := opt.Request()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Copy data
	if _, err = io.Copy(localFile, res.Body); err != nil {
		return err
	}

	return nil
}

// Create request and calculate MD5 for body response
func (opt *RequestOptions) MD5() (string, error) {
	res, err := opt.Request()
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	sumWriter := md5.New()
	if _, err = io.Copy(sumWriter, res.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(sumWriter.Sum(nil)), nil
}

// Create request and calculate SHA1 for body response, recomends use SHA256
func (opt *RequestOptions) SHA1() (string, error) {
	res, err := opt.Request()
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	sumWriter := sha1.New()
	if _, err = io.Copy(sumWriter, res.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(sumWriter.Sum(nil)), nil
}

// Create request and calculate SHA256 for body response
func (opt *RequestOptions) SHA256() (string, error) {
	res, err := opt.Request()
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	sumWriter := sha256.New()
	if _, err = io.Copy(sumWriter, res.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(sumWriter.Sum(nil)), nil
}