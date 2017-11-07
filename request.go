package fetcher

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

const (
	// ContentTypeJSON = "application/json"
	ContentTypeJSON = "application/json"

	// ContentTypeGob = "application/gob"
	ContentTypeGob = "application/gob"

	// ContentTypeXML = "application/xml"
	ContentTypeXML = "application/xml"

	// ContentTypeHeader = "Content-Type"
	ContentTypeHeader = "Content-Type"

	// AcceptHeader = "Accept"
	AcceptHeader = "Accept"
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
	cookies []*http.Cookie

	// BasicAuth options
	optBasicAuth bool
	username     string
	password     string

	// multipart form details
	optMultiPartForm bool
	multiPartFormErr error

	// append using RequestWithAfterDoFunc option
	afterDoFuncs []func(req *Request, resp *Response) error

	// convenience option for context cancellation
	deadline    time.Time
	clientTrace *httptrace.ClientTrace

	// retry config
	maxAttempts     int
	backoffStrategy backoffStrategy

	errorLogFunc LogFunc
	debugLogFunc LogFunc
}

// NewRequest returns a new Request with the given method/url and options executed
func (cl *Client) NewRequest(c context.Context, method, url string, opts ...RequestOption) (*Request, error) {
	req := &Request{
		method:          method,
		url:             url,
		maxAttempts:     1,
		headers:         map[string]string{},
		backoffStrategy: defaultBackoffStrategy,
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

	// add cookies
	for _, cookie := range req.cookies {
		req.request.AddCookie(cookie)
	}

	// set BasicAuth
	if req.optBasicAuth {
		req.request.SetBasicAuth(req.username, req.password)
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

// RequestWithReaderMultipartPayload takes a filepath, opens the file and adds it to the request
func RequestWithReaderMultipartPayload(filename string, data io.Reader) RequestOption {
	return func(c context.Context, req *Request) error {
		req.multipartPayload("file", filename, data)
		return nil
	}
}

// RequestWithFilepathMultipartPayload takes a filepath, opens the file and adds it to the request
func RequestWithFilepathMultipartPayload(filepath string) RequestOption {
	return func(c context.Context, req *Request) error {
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}

		fi, err := f.Stat()
		if err != nil {
			return err
		}

		req.multipartPayload("file", fi.Name(), f)
		return nil
	}
}

// TODO: this still buffers internally- see https://groups.google.com/forum/#!topic/golang-nuts/Zjg5l4nKcQ0
func (req *Request) multipartPayload(fieldname, filename string, data io.Reader) {
	// create a pipe to connect the data reader to the request payload
	pipeReader, pipeWriter := io.Pipe()
	mpw := multipart.NewWriter(pipeWriter)

	// set multipart request options
	req.optMultiPartForm = true
	req.payload = pipeReader
	req.headers["Content-Type"] = mpw.FormDataContentType()

	go func() {
		var err error
		var part io.Writer
		defer pipeWriter.Close()
		if closer, ok := data.(io.Closer); ok {
			defer closer.Close()
		}

		if part, err = mpw.CreateFormFile(fieldname, filename); err != nil {
			req.multiPartFormErr = err
			req.errorf("mpw.CreateFormFile failed: %s", err.Error())
			return
		}

		if _, err = io.Copy(part, data); err != nil {
			req.multiPartFormErr = err
			req.errorf("io.Copy failed: %s", err.Error())
			return
		}

		if err = mpw.Close(); err != nil {
			req.multiPartFormErr = err
			req.errorf("mpw.Close failed: %s", err.Error())
			return
		}
	}()
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

// RequestWithDefaultBackoff uses ExponentialJitterBackoff with min: 1s and max: 30s
func RequestWithDefaultBackoff() RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = defaultBackoffStrategy
		return nil
	}
}

// RequestWithNoBackoff waits delay duration on each retry, regardless of attempt number
func RequestWithNoBackoff(delay time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = noBackoff{
			delay: delay,
		}
		return nil
	}
}

// RequestWithLinearBackoff increases its delay by interval duration on each attempt
func RequestWithLinearBackoff(interval, min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = linearBackoff{
			min:       min,
			max:       max,
			interval:  interval,
			useJitter: false,
		}
		return nil
	}
}

// RequestWithLinearJitterBackoff increases its delay by interval duration on each attempt,
// with the each successive interval adjusted +/- 0-33%
func RequestWithLinearJitterBackoff(interval, min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = linearBackoff{
			min:       min,
			max:       max,
			interval:  interval,
			useJitter: true,
		}
		return nil
	}
}

// RequestWithExponentialBackoff multiplies the min duration by 2^(attempt number - 1), doubling the delay on each attempt
func RequestWithExponentialBackoff(min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = exponentialBackoff{
			min:       min,
			max:       max,
			useJitter: false,
		}
		return nil
	}
}

// RequestWithExponentialJitterBackoff multiplies the min duration by 2^(attempt number - 1), doubling the delay on each attempt
// with the each successive interval adjusted +/- 0-33%
func RequestWithExponentialJitterBackoff(min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = exponentialBackoff{
			min:       min,
			max:       max,
			useJitter: true,
		}
		return nil
	}
}

// RequestWithTimeout is a convenience function around context.WithTimeout
func RequestWithTimeout(timeout time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.deadline = time.Now().Add(timeout)
		return nil
	}
}

// RequestWithDeadline is a convenience function around context.WithDeadline
func RequestWithDeadline(deadline time.Time) RequestOption {
	return func(c context.Context, req *Request) error {
		req.deadline = deadline
		return nil
	}
}

// RequestWithClientTrace is a convenience function around httptrace.WithClientTrace
func RequestWithClientTrace(clientTrace *httptrace.ClientTrace) RequestOption {
	return func(c context.Context, req *Request) error {
		req.clientTrace = clientTrace
		return nil
	}
}

// RequestWithCookie adds a single cookie to the request
func RequestWithCookie(cookie *http.Cookie) RequestOption {
	return func(c context.Context, req *Request) error {
		req.cookies = append(req.cookies, cookie)
		return nil
	}
}

// RequestWithCookies adds a slice of cookies to the request
func RequestWithCookies(cookies []*http.Cookie) RequestOption {
	return func(c context.Context, req *Request) error {
		req.cookies = append(req.cookies, cookies...)
		return nil
	}
}

// RequestWithBasicAuth sets HTTP Basic Authentication authorization header
func RequestWithBasicAuth(username, password string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.optBasicAuth = true
		req.username = username
		req.password = password
		return nil
	}
}
