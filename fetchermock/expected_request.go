package fetchermock

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/nozzle/fetcher"
)

// ExpectedRequest
type ExpectedRequest struct {
	requestOptions []fetcher.RequestOption
	request        *fetcher.Request
	response       *fetcher.Response
	err            error

	wasMet bool

	// response
	responseBodyReader io.Reader
	responseStatusCode int
	responseStatus     string
	responseHeaders    map[string]string
}

// ExpectRequest creates an ExpectedRequest and adds it to the cl.expectedRequests
func (cl *Client) ExpectRequest(c context.Context, method, url string, opts ...ExpectedRequestOption) error {
	expReq := &ExpectedRequest{responseHeaders: map[string]string{}}

	// execute all options
	var err error
	for _, opt := range opts {
		if err = opt(c, expReq); err != nil {
			return err
		}
	}

	// create the request that will be matched with the executed request
	expReq.request, err = fetcher.NewRequest(c, method, url, expReq.requestOptions...)
	if err != nil {
		return err
	}

	// create the expected response
	expReq.response = fetcher.NewResponse(c, expReq.request, mockHttpResponse(c, expReq))

	// add the ExpectedRequest to the ExpectedRequests for the Client
	cl.expectedRequests = append(cl.expectedRequests, expReq)

	return nil
}

// ExpectedRequestOption is a func to configure optional settings for an ExpectedRequest
type ExpectedRequestOption func(c context.Context, expReq *ExpectedRequest) error

func WithRequestOptions(opts ...fetcher.RequestOption) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.requestOptions = opts
		return nil
	}
}

func WithResponseStatusCode(code int) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.responseStatusCode = code
		return nil
	}
}

func WithResponseStatus(status string) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.responseStatus = status
		return nil
	}
}

func WithResponseBodyBytes(b []byte) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.responseBodyReader = bytes.NewReader(b)
		return nil
	}
}

func WithResponseBodyReader(r io.Reader) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.responseBodyReader = r
		return nil
	}
}

func WithResponseHeader(key, value string) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.responseHeaders[key] = value
		return nil
	}
}

func WithResponseError(err error) ExpectedRequestOption {
	return func(c context.Context, expReq *ExpectedRequest) error {
		expReq.err = err
		return nil
	}
}

func mockHttpResponse(c context.Context, expReq *ExpectedRequest) *http.Response {
	resp := &http.Response{Header: http.Header(map[string][]string{})}
	resp.Body = ioutil.NopCloser(expReq.responseBodyReader)
	for key, value := range expReq.responseHeaders {
		resp.Header.Set(key, value)
	}
	return resp
}
