package fetcher

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"io"
)

// DecodeFunc allows users to provide a custom decoder to use with Decode
type DecodeFunc func(io.Reader, interface{}) error

func jsonDecodeFunc(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func gobDecodeFunc(r io.Reader, v interface{}) error {
	return gob.NewDecoder(r).Decode(v)
}

func xmlDecodeFunc(r io.Reader, v interface{}) error {
	return xml.NewDecoder(r).Decode(v)
}

// DecodeOption is a func to configure optional Response settings
type DecodeOption func(c context.Context, resp *Response) error

// WithJSONBody json decodes the body of the Response
func WithJSONBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = jsonDecodeFunc
		return nil
	}
}

// WithGobBody gob decodes the body of the Response
func WithGobBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = gobDecodeFunc
		return nil
	}
}

// WithXMLBody xml decodes the body of the Response
func WithXMLBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = xmlDecodeFunc
		return nil
	}
}

// WithCopiedBody makes a copy of the body available in the response.
// This is helpful if you anticipate the decode failing and want to do a full
// dump of the response.
func WithCopiedBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		buf := getBuffer()
		resp.body = io.TeeReader(resp.response.Body, buf)
		resp.copiedBody = buf
		resp.keepBody = true
		return nil
	}
}

// WithCustomFunc uses the provided DecodeFunc to Decode the response
func WithCustomFunc(decodeFunc DecodeFunc) DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = decodeFunc
		return nil
	}
}
