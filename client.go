package fetcher

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"time"
)

var _ Fetcher = (*Client)(nil)

// Client implements Fetcher interface and is required to execute a Request
type Client struct {
	client *http.Client

	keepAlive        time.Duration
	handshakeTimeout time.Duration

	errorLogFunc LogFunc
	debugLogFunc LogFunc
}

// NewClient returns a new Client with the given options executed
func NewClient(c context.Context, opts ...ClientOption) (*Client, error) {
	cl := &Client{
		keepAlive:        60 * time.Second,
		handshakeTimeout: 10 * time.Second,
	}

	var err error

	// execute all options
	for _, opt := range opts {
		if err = opt(c, cl); err != nil {
			return nil, err
		}
	}

	cl.setClient()

	return cl, nil
}

// Do uses the client receiver to execute the provided request
func (cl *Client) Do(c context.Context, req *Request) (*Response, error) {
	// if the context has been canceled or the deadline exceeded, don't start the request
	if c.Err() != nil {
		return nil, c.Err()
	}

	// if per request loggers haven't been set, inherit from the client
	if cl.debugLogFunc != nil && req.debugLogFunc == nil {
		req.debugLogFunc = cl.debugLogFunc
		req.debugf("request using client debugLogFunc")
	}
	if cl.errorLogFunc != nil && req.errorLogFunc == nil {
		req.errorLogFunc = cl.errorLogFunc
		req.debugf("request using client errorLogFunc")
	}

	// set the context deadline if one was provided in the request options
	if !req.deadline.IsZero() {
		req.debugf("setting context deadline to %s", req.deadline)
		var cancelFunc context.CancelFunc
		c, cancelFunc = context.WithDeadline(c, req.deadline)
		defer cancelFunc()
	}

	req.client = cl

	reqc := req.request.WithContext(c)

	if buf, ok := req.payload.(*bytes.Buffer); ok {
		defer putBuffer(buf)
	}

	resp := &Response{}
	var err error
	for i := 1; i <= req.maxAttempts; i++ {
		req.debugf("request attempt #%d", i)
		resp.response, err = cl.client.Do(reqc)
		if err != nil {
			return nil, err
		}

		// further attempts will be made only on 500+ status codes
		// NOTE: the error returned from cl.client.Do(reqc) only contains scenarios regarding
		// a bad request given, or a response with Location header missing or bad
		if resp.response.StatusCode < 500 {
			req.debugf("status code %d < 500, exiting retry loop", resp.response.StatusCode)
			break
		}

		// break out of the retry loop if this is the last attempt, so we don't close the response body
		// or sleep unnecessarily
		if i == req.maxAttempts {
			req.debugf("max attempts (%d) reached, exiting retry loop", req.maxAttempts)
			break
		}

		// close the response body before we lose our reference to it
		if err = resp.response.Body.Close(); err != nil {
			req.errorf(err.Error())
			return nil, err
		}

		// wait before retrying, returning early if the context is cancelled
		delay := req.backoffStrategy.waitDuration(i)
		req.debugf("waiting %s before next retry", delay)

		select {
		case <-time.After(delay):
		case <-c.Done():
			req.debugf("context cancelled during backoff delay")
			return nil, c.Err()
		}
	}

	// execute all afterDoFuncs
	for _, afterDo := range req.afterDoFuncs {
		if err = afterDo(req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// Get is a helper func for Do, setting the Method internally
func (cl *Client) Get(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodGet, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Head is a helper func for Do, setting the Method internally
func (cl *Client) Head(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodHead, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Post is a helper func for Do, setting the Method internally
func (cl *Client) Post(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPost, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Put is a helper func for Do, setting the Method internally
func (cl *Client) Put(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPut, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Patch is a helper func for Do, setting the Method internally
func (cl *Client) Patch(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPatch, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Delete is a helper func for Do, setting the Method internally
func (cl *Client) Delete(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodDelete, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// ClientOption is a func to configure optional Client settings
type ClientOption func(c context.Context, cl *Client) error

// ClientWithKeepAlive is a ClientOption that sets the cl.keepAlive field to the given duration
func ClientWithKeepAlive(dur time.Duration) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.keepAlive = dur
		return nil
	}
}

// ClientWithHandshakeTimeout is a ClientOption that sets the cl.handshakeTimeout field to the given duration
func ClientWithHandshakeTimeout(dur time.Duration) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.handshakeTimeout = dur
		return nil
	}
}

// setClient creates the standard http.Client using the settings in the given Client
func (cl *Client) setClient() {
	cl.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				KeepAlive: cl.keepAlive,
			}).Dial,
			TLSHandshakeTimeout: cl.handshakeTimeout,
		},
	}
}
