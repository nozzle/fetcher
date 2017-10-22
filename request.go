package fetcher

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ContentTypeJSON   = "application/json"
	ContentTypeGob    = "application/gob"
	ContentTypeHeader = "Content-Type"
	AcceptHeader      = "Accept"
)

// Request contains the data for a http.Request to be created
type Request struct {
	client  *Client
	request *http.Request

	// set through options
	method  string
	url     string
	payload io.Reader
	headers map[string]string

	// append using RequestWithAfterDoFunc option
	afterDoFuncs []func(req *Request, resp *Response) error

	// response config
	maxAttempts int
}

// NewRequest returns a new Request with the given method/url and options executed
func NewRequest(c context.Context, method, url string, opts ...RequestOption) (*Request, error) {
	req := &Request{
		method:      method,
		url:         url,
		maxAttempts: 1,
		headers:     map[string]string{},
	}
	var err error

	// execute all options
	for _, opt := range opts {
		if err = opt(c, req); err != nil {
			return nil, err
		}
	}

	// setDefaultRequestOptions(req)
	req.request, err = http.NewRequest(req.method, req.url, req.payload)
	if err != nil {
		return nil, err
	}

	// add the headers
	for key, value := range req.headers {
		req.request.Header.Add(key, value)
	}

	req.request.Close = false

	return req, nil
}

// String is a stringer for Request
func (req Request) String() string {
	return fmt.Sprintf("method:%s | url:%s | maxAttempts:%d | headers:%s",
		req.method,
		req.url,
		req.maxAttempts,
		req.headers,
	)
}

// Equal compares the request with another request
// If not equal, a string is returned with first field found different
// used by fetchermock
func (req *Request) Equal(reqComp *Request) (bool, string) {
	if reqComp == nil {
		return false, "comparison Request is nil"
	}
	if req.method != reqComp.method {
		return false, fmt.Sprintf("method: %s != %s", req.method, reqComp.method)
	}
	if req.url != reqComp.url {
		return false, fmt.Sprintf("url: %s != %s", req.url, reqComp.url)
	}
	if req.maxAttempts != reqComp.maxAttempts {
		return false, fmt.Sprintf("maxAttempts: %d != %d", req.maxAttempts, reqComp.maxAttempts)
	}
	for key, value := range req.headers {
		if _, ok := reqComp.headers[key]; !ok {
			return false, fmt.Sprintf("headers-key: '%s' not found", key)
		}
		if value != reqComp.headers[key] {
			return false, fmt.Sprintf("headers-value: key '%s' | %s != %s", key, value, reqComp.headers[key])
		}
	}
	return true, ""
}

// RequestOption is a func to configure optional Request settings
type RequestOption func(c context.Context, req *Request) error

// RequestWithJSONPayload json marshals the payload for the Request
// and sets the content-type header to application/json
func RequestWithJSONPayload(payload interface{}) RequestOption {
	return func(c context.Context, req *Request) error {
		if payload == nil {
			return nil
		}
		req.headers[AcceptHeader] = ContentTypeJSON
		req.headers[ContentTypeHeader] = ContentTypeJSON
		buf := getBuffer()
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return err
		}
		req.payload = buf
		return nil
	}
}

// RequestWithGobPayload gob encodes the payload for the Request
// and sets the content-type header to application/gob
func RequestWithGobPayload(payload interface{}) RequestOption {
	return func(c context.Context, req *Request) error {
		if payload == nil {
			return nil
		}
		buf := getBuffer()
		if err := gob.NewEncoder(buf).Encode(payload); err != nil {
			return err
		}
		req.payload = buf
		return nil
	}
}

// RequestWithBytesPayload sets the given payload for the Request
func RequestWithBytesPayload(payload []byte) RequestOption {
	return func(c context.Context, req *Request) error {
		req.payload = bytes.NewReader(payload)
		return nil
	}
}

// RequestWithReaderPayload sets the given payload for the Request
func RequestWithReaderPayload(payload io.Reader) RequestOption {
	return func(c context.Context, req *Request) error {
		req.payload = payload
		return nil
	}
}

// RequestWithHeader adds the given key/value combo to the Request headers
func RequestWithHeader(key, value string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.headers[key] = value
		return nil
	}
}

// RequestWithAcceptJSONHeader adds Accept: application/json to the Request headers
func RequestWithAcceptJSONHeader() RequestOption {
	return func(c context.Context, req *Request) error {
		req.headers[AcceptHeader] = ContentTypeJSON
		return nil
	}
}

// RequestWithMaxAttempts sets the max number of times to attempt the Request on 5xx status code
// must be at least 1
func RequestWithMaxAttempts(maxAttempts int) RequestOption {
	return func(c context.Context, req *Request) error {
		if maxAttempts < 1 {
			maxAttempts = 1
		}
		req.maxAttempts = maxAttempts
		return nil
	}
}

// RequestWithAfterDoFunc allows user-defined functions to access Request and Response (read-only)
func RequestWithAfterDoFunc(afterDoFunc func(req *Request, resp *Response) error) RequestOption {
	return func(c context.Context, req *Request) error {
		req.afterDoFuncs = append(req.afterDoFuncs, afterDoFunc)
		return nil
	}
}
