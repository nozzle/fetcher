package fetchermock

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nozzle/fetcher"
)

var _ fetcher.Fetcher = (*Client)(nil)

type Client struct {
	expectedRequests []*ExpectedRequest

	withExpectationsInOrder bool
	expectationsMet         bool
}

// NewClient returns a new Client with the given options executed
func NewClient(c context.Context, opts ...ClientOption) (*Client, error) {
	cl := &Client{
		expectedRequests:        []*ExpectedRequest{},
		withExpectationsInOrder: true,
		expectationsMet:         false,
	}

	// execute all options
	var err error
	for _, opt := range opts {
		if err = opt(c, cl); err != nil {
			return nil, err
		}
	}

	return cl, nil
}

func (cl *Client) Do(c context.Context, req *fetcher.Request) (*fetcher.Response, error) {
	// if the context has been canceled or the deadline exceeded, don't start the request
	if c.Err() != nil {
		return nil, c.Err()
	}

	// find the expected request in cl.expectedRequests
	var expReqWasMet bool
	var metIdx int
	var equal bool
	var info string
	for i := range cl.expectedRequests {
		if cl.expectedRequests[i].wasMet {
			continue
		}

		// compare the expectations to the actual request
		equal, info = cl.expectedRequests[i].request.Equal(req)
		if equal {
			cl.expectedRequests[i].wasMet = true
			expReqWasMet = true
			metIdx = i
			break
		}

		// if the expectations are to be in order, and this expectation wasn't met, error out
		if cl.withExpectationsInOrder && !cl.expectedRequests[i].wasMet {
			return nil, fmt.Errorf("ExpectedRequest did not match fetcher.Request | info: %s", info)
		}
	}

	// if not met, error out
	if !expReqWasMet {
		return nil, fmt.Errorf("Request did not match any ExpectedRequests | %s", req.String())
	}

	// if met, return the expReq.response
	if cl.metCount() == len(cl.expectedRequests) {
		cl.expectationsMet = true
	}

	return cl.expectedRequests[metIdx].response, nil
}

func (cl *Client) Get(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodGet, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) Head(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodHead, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) Post(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodPost, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) Put(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodPut, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) Patch(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodPatch, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) Delete(c context.Context, url string, opts ...fetcher.RequestOption) (*fetcher.Response, error) {
	req, err := fetcher.NewRequest(c, http.MethodDelete, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

func (cl *Client) UnmetExpectations() []*ExpectedRequest {
	unmet := make([]*ExpectedRequest, 0, len(cl.expectedRequests)-cl.metCount())
	for i := range cl.expectedRequests {
		if !cl.expectedRequests[i].wasMet {
			unmet = append(unmet, cl.expectedRequests[i])
		}
	}
	return unmet
}

func (cl *Client) metCount() int {
	metCount := 0
	for i := range cl.expectedRequests {
		if cl.expectedRequests[i].wasMet {
			metCount++
		}
	}
	return metCount
}

// ClientOption is a func to configure optional Client settings
type ClientOption func(c context.Context, cl *Client) error

// ClientWithExpectationsInOrder sets the cl.withExpectationsInOrder value
func ClientWithExpectationsInOrder(inOrder bool) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.withExpectationsInOrder = inOrder
		return nil
	}
}
