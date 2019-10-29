package fetcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

// Response is returned after executing client.Do
type Response struct {
	request    *Request
	response   *http.Response
	body       io.Reader
	copiedBody *bytes.Buffer

	// set through Options
	keepBody bool

	// used by Close()
	bodyClosed bool

	decodeFunc DecodeFunc
}

// NewResponse returns a Response with the given Request and http.Response
func NewResponse(c context.Context, req *Request, resp *http.Response) *Response {
	return &Response{
		request:  req,
		response: resp,
		body:     resp.Body,
	}
}

// Decode decodes the resp.response.Body into the given object (v) using the specified decoder
// NOTE: v is assumed to be a pointer
func (resp *Response) Decode(c context.Context, v interface{}, opts ...DecodeOption) error {
	// execute all options
	var err error
	for _, opt := range opts {
		if err = opt(c, resp); err != nil {
			return err
		}
	}

	// auto-set the decoder based on the response header if one hasn't been specified
	if resp.decodeFunc == nil {
		resp.decodeFunc = resp.detectDecoder()
	}

	defer resp.response.Body.Close()

	if resp.decodeFunc == nil {
		return errors.New("no valid decoder specified")
	}

	return resp.decodeFunc(resp.body, v)
}

// detectDecoder auto-selects a decoder based on the response header
func (resp *Response) detectDecoder() DecodeFunc {
	switch resp.response.Header.Get(ContentTypeHeader) {
	case ContentTypeJSON:
		resp.request.debugf("json encoding detected")
		return jsonDecodeFunc

	case ContentTypeGob:
		resp.request.debugf("gob encoding detected")
		return gobDecodeFunc

	case ContentTypeXML:
		resp.request.debugf("xml encoding detected")
		return xmlDecodeFunc
	}

	return nil
}

// Bytes reads the body into a buffer and then returns the bytes
// returns error based on resp.response.Body.Close()
func (resp *Response) Bytes() ([]byte, error) {
	if resp.copiedBody != nil {
		return resp.copiedBody.Bytes(), nil
	}
	buf := getBuffer()
	buf.ReadFrom(resp.response.Body)
	if err := resp.response.Body.Close(); err != nil {
		return nil, err
	}
	resp.bodyClosed = true
	resp.copiedBody = buf
	return buf.Bytes(), nil
}

// MustBytes reads the body into a buffer and then returns the bytes
func (resp *Response) MustBytes() []byte {
	bts, err := resp.Bytes()
	if err != nil {
		resp.request.errorf("MustBytes error: %s", err.Error())
	}
	return bts
}

// Body returns the resp.response.Body as io.Reader
// NOTE: original io.ReadCloser body is closed when Close is called by the user
func (resp *Response) Body() io.Reader {
	if resp.keepBody && resp.copiedBody != nil {
		return resp.copiedBody
	}
	return resp.response.Body
}

// Close handles any needed clean-up after the user is done with the Response object
func (resp *Response) Close() error {
	if resp.keepBody && resp.copiedBody != nil {
		putBuffer(resp.copiedBody)
	}
	if resp.bodyClosed {
		return nil
	}
	if err := resp.response.Body.Close(); err != io.EOF {
		return err
	}
	return nil
}

// StatusCode exports resp.StatusCode
func (resp *Response) StatusCode() int {
	return resp.response.StatusCode
}

// Status exports resp.Status
func (resp *Response) Status() string {
	return resp.response.Status
}

// FinalURL returns the final URL from resp.Request
func (resp *Response) FinalURL() *url.URL {
	return resp.response.Request.URL
}

// RequestURL returns the resp.request.url
func (resp *Response) RequestURL() string {
	return resp.request.url
}

// ContentType returns the Content-Type header value of the Response
func (resp *Response) ContentType() string {
	return resp.response.Header.Get("Content-Type")
}
