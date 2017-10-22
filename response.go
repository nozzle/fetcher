package fetcher

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Response is returned after executing client.Do
type Response struct {
	request    *Request
	response   *http.Response
	copiedBody io.Reader

	// set through Options
	contentType string
	keepBody    bool

	// used by Close()
	bodyClosed bool
}

// NewResponse returns a Response with the given Request and http.Response
func NewResponse(c context.Context, req *Request, resp *http.Response) *Response {
	return &Response{
		request:  req,
		response: resp,
	}
}

// Decode decodes the resp.response.Body into the given object (v) using the specified decoder
// NOTE: v is assumed to be a pointer
func (resp *Response) Decode(c context.Context, v interface{}, opts ...DecodeOption) error {
	// auto-set the contentType based on the response header
	// NOTE: will be overwritten with DecodeWithJSON or DecodeWithGob options
	resp.contentType = resp.response.Header.Get(ContentTypeHeader)

	// execute all options
	var err error
	for _, opt := range opts {
		if err = opt(c, resp); err != nil {
			return err
		}
	}

	defer resp.response.Body.Close()
	var r io.Reader = resp.response.Body
	if resp.keepBody {
		buf := getBuffer()
		r = io.TeeReader(resp.response.Body, buf)
		resp.copiedBody = buf
	}

	switch resp.contentType {
	case ContentTypeJSON:
		if err = json.NewDecoder(r).Decode(v); err != nil {
			return err
		}

	case ContentTypeGob:
		if err = gob.NewDecoder(r).Decode(v); err != nil {
			return err
		}

	default:
		return errors.New("unrecognized contentType")
	}

	return nil
}

// Bytes reads the body into a buffer and then returns the bytes
func (resp *Response) Bytes() ([]byte, error) {
	buf := getBuffer()
	buf.ReadFrom(resp.response.Body)
	if err := resp.response.Body.Close(); err != nil {
		return nil, err
	}
	resp.bodyClosed = true
	resp.copiedBody = buf
	return buf.Bytes(), nil
}

// Body returns the resp.response.Body as io.Reader
// NOTE: original io.ReadCloser body is closed when Close is called by the user
func (resp *Response) Body() io.Reader {
	if resp.keepBody {
		return resp.copiedBody
	}
	return resp.response.Body
}

// Close handles any needed clean-up after the user is done with the Response object
func (resp *Response) Close() error {
	if resp.keepBody {
		if buf, ok := resp.copiedBody.(*bytes.Buffer); ok {
			putBuffer(buf)
		}
	}
	if resp.bodyClosed {
		return nil
	}
	if err := resp.response.Body.Close(); err != io.EOF {
		return err
	}
	return nil
}

func (resp *Response) StatusCode() int {
	return resp.response.StatusCode
}

func (resp *Response) Status() string {
	return resp.response.Status
}

// DecodeOption is a func to configure optional Response settings
type DecodeOption func(c context.Context, resp *Response) error

// DecodeWithJSON json decodes the body of the Response
func DecodeWithJSON() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.contentType = ContentTypeJSON
		return nil
	}
}

// DecodeWithGob gob decodes the body of the Response
func DecodeWithGob() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.contentType = ContentTypeGob
		return nil
	}
}

// DecodeWithCopiedBody gob decodes the body of the Response
func DecodeWithCopiedBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.keepBody = true
		return nil
	}
}
