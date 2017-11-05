package fetcher

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"io"
)

// Decoder is an interface for providing a custom decoder
type DecodeFunc func(io.Reader, interface{}) error

func jsonDecodeFunc(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func gobDecodeFunc(r io.Reader, v interface{}) error {
	return gob.NewDecoder(r).Decode(v)
}

// DecodeOption is a func to configure optional Response settings
type DecodeOption func(c context.Context, resp *Response) error

// DecodeWithJSON json decodes the body of the Response
func DecodeWithJSON() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = jsonDecodeFunc
		return nil
	}
}

// DecodeWithGob gob decodes the body of the Response
func DecodeWithGob() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = gobDecodeFunc
		return nil
	}
}

// DecodeWithXML xml decodes the body of the Response
func DecodeWithXML() DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = xmlDecodeFunc
		return nil
	}
}

// DecodeWithCopiedBody makes a copy of the body available in the response.
// This is helpful if you anticipate the decode failing and want to do a full
// dump of the response.
func DecodeWithCopiedBody() DecodeOption {
	return func(c context.Context, resp *Response) error {
		buf := getBuffer()
		resp.body = io.TeeReader(resp.response.Body, buf)
		resp.copiedBody = buf
		resp.keepBody = true
		return nil
	}
}

// DecodeWithCustomFunc uses the provided DecodeFunc to Decode the response
func DecodeWithCustomFunc(decodeFunc DecodeFunc) DecodeOption {
	return func(c context.Context, resp *Response) error {
		resp.decodeFunc = decodeFunc
		return nil
	}
}
