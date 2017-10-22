package fetcher

import "context"

// Fetcher is the interface that a Client will need to implement in order to execute a Request
type Fetcher interface {
	Do(c context.Context, req *Request) (*Response, error)
	Get(c context.Context, url string, opts ...RequestOption) (*Response, error)
	Head(c context.Context, url string, opts ...RequestOption) (*Response, error)
	Post(c context.Context, url string, opts ...RequestOption) (*Response, error)
	Put(c context.Context, url string, opts ...RequestOption) (*Response, error)
	Patch(c context.Context, url string, opts ...RequestOption) (*Response, error)
	Delete(c context.Context, url string, opts ...RequestOption) (*Response, error)
}
