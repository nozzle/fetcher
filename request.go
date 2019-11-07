package fetcher

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// ContentTypeJSON = "application/json"
	ContentTypeJSON = "application/json"

	// ContentTypeGob = "application/gob"
	ContentTypeGob = "application/gob"

	// ContentTypeXML = "application/xml"
	ContentTypeXML = "application/xml"

	// ContentTypeURLEncoded = "application/x-www-form-urlencoded"
	ContentTypeURLEncoded = "application/x-www-form-urlencoded"

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
	params  []param
	headers []header
	cookies []*http.Cookie

	// BasicAuth options
	optBasicAuth bool
	username     string
	password     string

	// multipart form details
	optMultiPartForm         bool
	multiPartFormFieldParams []param
	multiPartFormErr         error

	// append using WithAfterDoFunc option
	afterDoFuncs []func(req *Request, resp *Response) error

	// convenience option for context cancellation
	deadline    time.Time
	clientTrace *httptrace.ClientTrace

	// retry config
	maxAttempts     int
	backoffStrategy backoffStrategy
	retryOnEOFError bool

	errorLogFunc LogFunc
	debugLogFunc LogFunc
}

// NewRequest returns a new Request with the given method/url and options executed
func (cl *Client) NewRequest(c context.Context, method, urlStr string, opts ...RequestOption) (*Request, error) {
	req := &Request{
		method:          method,
		url:             urlStr,
		maxAttempts:     1,
		backoffStrategy: defaultBackoffStrategy,
	}
	var err error

	// prepend options with cl.parentRequestOptions
	opts = append(cl.parentRequestOptions, opts...)

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
	for i := range req.headers {
		req.request.Header.Add(req.headers[i].key, req.headers[i].value)
	}

	// add the params and write to the URL
	if len(req.params) > 0 {
		params := url.Values{}
		for i := range req.params {
			params.Add(req.params[i].key, req.params[i].value)
		}
		req.request.URL.RawQuery = params.Encode()
		req.url = req.request.URL.String()
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
	var payload []byte
	switch v := req.payload.(type) {
	case *bytes.Buffer:
		payload = v.Bytes()
	}
	return fmt.Sprintf("method:%s | url:%s | maxAttempts:%d | headers:%s | payload (string):'%s'",
		req.method,
		req.url,
		req.maxAttempts,
		req.headers,
		string(payload),
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
	for i := range req.headers {
		if req.headers[i] != reqComp.headers[i] {
			return false, fmt.Sprintf("headers[%d]: %s != %s", i, req.headers[i], reqComp.headers[i])
		}
	}

	if req.payload != nil && reqComp.payload != nil {
		reqBody, err := ioutil.ReadAll(req.payload)
		if err != nil {
			return false, fmt.Sprintf("couldn't read body %s", err)
		}

		reqCompBody, err := ioutil.ReadAll(reqComp.payload)
		if err != nil {
			return false, fmt.Sprintf("couldn't read body %s", err)
		}

		if len(reqBody) != len(reqCompBody) {
			return false, fmt.Sprintf("bodies don't match got %s expected %s", reqBody, reqCompBody)
		}

		for i := range reqBody {
			if reqBody[i] != reqCompBody[i] {
				return false, fmt.Sprintf("bodies don't match got %s expected %s", reqBody, reqCompBody)
			}
		}
	}

	return true, ""
}

type header struct {
	key, value string
}

func newHeader(key, value string) header {
	return header{
		key:   key,
		value: value,
	}
}

type param struct {
	key, value string
}

func newParam(key, value string) param {
	return param{
		key:   key,
		value: value,
	}
}

// RequestOption is a func to configure optional Request settings
type RequestOption func(c context.Context, req *Request) error

// WithBaseURL prepends the req.url with the given baseURL
func WithBaseURL(baseURL string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.url = baseURL + req.url
		return nil
	}
}

// WithJSONPayload json marshals the payload for the Request
// and sets the content-type and accept header to application/json
func WithJSONPayload(payload interface{}) RequestOption {
	return func(c context.Context, req *Request) error {
		if payload == nil {
			return nil
		}
		req.headers = append(req.headers, newHeader(AcceptHeader, ContentTypeJSON))
		req.headers = append(req.headers, newHeader(ContentTypeHeader, ContentTypeJSON))
		buf := getBuffer()
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return err
		}
		req.payload = buf
		return nil
	}
}

// WithGobPayload gob encodes the payload for the Request
// and sets the content-type and accept header to application/gob
func WithGobPayload(payload interface{}) RequestOption {
	return func(c context.Context, req *Request) error {
		if payload == nil {
			return nil
		}
		req.headers = append(req.headers, newHeader(AcceptHeader, ContentTypeGob))
		req.headers = append(req.headers, newHeader(ContentTypeHeader, ContentTypeGob))
		buf := getBuffer()
		if err := gob.NewEncoder(buf).Encode(payload); err != nil {
			return err
		}
		req.payload = buf
		return nil
	}
}

// WithURLEncodedPayload encodes the payload for the Request
// and sets the content-type header to application/x-www-form-urlencoded
func WithURLEncodedPayload(payload url.Values) RequestOption {
	return func(c context.Context, req *Request) error {
		if payload == nil {
			return nil
		}
		buf := getBuffer()
		buf.WriteString(payload.Encode())
		req.headers = append(req.headers, newHeader(ContentTypeHeader, ContentTypeURLEncoded))
		req.payload = buf
		return nil
	}
}

// WithParam adds parameter value to be encoded for the Request
func WithParam(key, value string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.params = append(req.params, newParam(key, value))
		return nil
	}
}

// WithBytesPayload sets the given payload for the Request
func WithBytesPayload(payload []byte) RequestOption {
	return func(c context.Context, req *Request) error {
		req.payload = bytes.NewReader(payload)
		return nil
	}
}

// WithRetryOnEOFError adds the io.EOF error to the retry loop
// The io.EOF error indicates sending on a broken connection (see https://github.com/golang/go/issues/8946 & https://github.com/golang/go/issues/5312)
// Including this option with a Request will allow fetcher to retry the request on io.EOF, in attempt to obtain a valid connection
func WithRetryOnEOFError() RequestOption {
	return func(c context.Context, req *Request) error {
		req.retryOnEOFError = true
		return nil
	}
}

// WithReaderMultipartField adds the fieldname and value to the multipart fields
func WithReaderMultipartField(fieldname, value string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.multiPartFormFieldParams = append(req.multiPartFormFieldParams, newParam(fieldname, value))
		return nil
	}
}

// WithReaderMultipartPayload takes a filepath, opens the file and adds it to the request with the fieldname
func WithReaderMultipartPayload(fieldname, filename string, data io.Reader) RequestOption {
	return func(c context.Context, req *Request) error {
		req.multipartPayload(fieldname, filename, data)
		return nil
	}
}

// WithFilepathMultipartPayload takes a filepath, opens the file and adds it to the request with the fieldname
func WithFilepathMultipartPayload(fieldname, filepath string) RequestOption {
	return func(c context.Context, req *Request) error {
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}

		fi, err := f.Stat()
		if err != nil {
			return err
		}

		req.multipartPayload(fieldname, fi.Name(), f)
		return nil
	}
}

// TODO: this still buffers internally - see https://groups.google.com/forum/#!topic/golang-nuts/Zjg5l4nKcQ0
func (req *Request) multipartPayload(fieldname, filename string, data io.Reader) {
	// create a pipe to connect the data reader to the request payload
	pipeReader, pipeWriter := io.Pipe()
	mpw := multipart.NewWriter(pipeWriter)

	// set multipart request options
	req.optMultiPartForm = true

	// set the multipart fields
	for i := range req.multiPartFormFieldParams {
		fldErr := mpw.WriteField(req.multiPartFormFieldParams[i].key, req.multiPartFormFieldParams[i].value)
		if fldErr != nil {
			req.multiPartFormErr = fldErr
			req.errorf("mpw.CreateFormFile failed: %s", fldErr.Error())
			return
		}
	}

	// set the payload
	req.payload = pipeReader
	req.headers = append(req.headers, newHeader(ContentTypeHeader, mpw.FormDataContentType()))

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

// isErrBreaking returns false if the given error is involved with an option called by the user
func (req *Request) isErrBreaking(err error) bool {
	switch {
	case strings.Contains(err.Error(), "read: connection reset by peer"),
		req.retryOnEOFError && err == io.EOF:
		return false
	default:
		return true
	}
}

// WithReaderPayload sets the given payload for the Request
func WithReaderPayload(payload io.Reader) RequestOption {
	return func(c context.Context, req *Request) error {
		req.payload = payload
		return nil
	}
}

// WithHeader adds the given key/value combo to the Request headers
func WithHeader(key, value string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.headers = append(req.headers, newHeader(key, value))
		return nil
	}
}

// WithAcceptJSONHeader adds Accept: application/json to the Request headers
func WithAcceptJSONHeader() RequestOption {
	return func(c context.Context, req *Request) error {
		req.headers = append(req.headers, newHeader(AcceptHeader, ContentTypeJSON))
		return nil
	}
}

// WithMaxAttempts sets the max number of times to attempt the Request on 5xx status code
// must be at least 1
func WithMaxAttempts(maxAttempts int) RequestOption {
	return func(c context.Context, req *Request) error {
		if maxAttempts < 1 {
			maxAttempts = 1
		}
		req.maxAttempts = maxAttempts
		return nil
	}
}

// WithAfterDoFunc allows user-defined functions to access Request and Response (read-only)
func WithAfterDoFunc(afterDoFunc func(req *Request, resp *Response) error) RequestOption {
	return func(c context.Context, req *Request) error {
		req.afterDoFuncs = append(req.afterDoFuncs, afterDoFunc)
		return nil
	}
}

// WithDefaultBackoff uses ExponentialJitterBackoff with min: 1s and max: 30s
func WithDefaultBackoff() RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = defaultBackoffStrategy
		return nil
	}
}

// WithNoBackoff waits delay duration on each retry, regardless of attempt number
func WithNoBackoff(delay time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = noBackoff{
			delay: delay,
		}
		return nil
	}
}

// WithLinearBackoff increases its delay by interval duration on each attempt
func WithLinearBackoff(interval, min, max time.Duration) RequestOption {
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

// WithLinearJitterBackoff increases its delay by interval duration on each attempt,
// with the each successive interval adjusted +/- 0-33%
func WithLinearJitterBackoff(interval, min, max time.Duration) RequestOption {
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

// WithExponentialBackoff multiplies the min duration by 2^(attempt number - 1), doubling the delay on each attempt
func WithExponentialBackoff(min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = exponentialBackoff{
			min:       min,
			max:       max,
			useJitter: false,
		}
		return nil
	}
}

// WithExponentialJitterBackoff multiplies the min duration by 2^(attempt number - 1), doubling the delay on each attempt
// with the each successive interval adjusted +/- 0-33%
func WithExponentialJitterBackoff(min, max time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.backoffStrategy = exponentialBackoff{
			min:       min,
			max:       max,
			useJitter: true,
		}
		return nil
	}
}

// WithTimeout is a convenience function around context.WithTimeout
func WithTimeout(timeout time.Duration) RequestOption {
	return func(c context.Context, req *Request) error {
		req.deadline = time.Now().Add(timeout)
		return nil
	}
}

// WithDeadline is a convenience function around context.WithDeadline
func WithDeadline(deadline time.Time) RequestOption {
	return func(c context.Context, req *Request) error {
		req.deadline = deadline
		return nil
	}
}

// WithClientTrace is a convenience function around httptrace.WithClientTrace
func WithClientTrace(clientTrace *httptrace.ClientTrace) RequestOption {
	return func(c context.Context, req *Request) error {
		req.clientTrace = clientTrace
		return nil
	}
}

// WithCookie adds a single cookie to the request
func WithCookie(cookie *http.Cookie) RequestOption {
	return func(c context.Context, req *Request) error {
		req.cookies = append(req.cookies, cookie)
		return nil
	}
}

// WithCookies adds a slice of cookies to the request
func WithCookies(cookies []*http.Cookie) RequestOption {
	return func(c context.Context, req *Request) error {
		req.cookies = append(req.cookies, cookies...)
		return nil
	}
}

// WithBasicAuth sets HTTP Basic Authentication authorization header
func WithBasicAuth(username, password string) RequestOption {
	return func(c context.Context, req *Request) error {
		req.optBasicAuth = true
		req.username = username
		req.password = password
		return nil
	}
}
