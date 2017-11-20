package fetcher

import "context"
import "fmt"

// LogFunc is a pluggable log function
type LogFunc func(string)

// WithClientDebugLogFunc pipes all debug logs to the supplied function
// All requests from this client inherit this logger
func WithClientDebugLogFunc(fn LogFunc) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.debugLogFunc = fn
		return nil
	}
}

// WithClientErrorLogFunc pipes all error logs to the supplied function
// All requests from this client inherit this logger
func WithClientErrorLogFunc(fn LogFunc) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.errorLogFunc = fn
		return nil
	}
}

// WithRequestDebugLogFunc pipes all debug logs to the supplied function
// This overrides and replaces the inherited client functions
func WithRequestDebugLogFunc(fn LogFunc) RequestOption {
	return func(c context.Context, req *Request) error {
		req.debugLogFunc = fn
		return nil
	}
}

// WithRequestErrorLogFunc pipes all error logs to the supplied function
// This overrides and replaces the inherited client functions
func WithRequestErrorLogFunc(fn LogFunc) RequestOption {
	return func(c context.Context, req *Request) error {
		req.errorLogFunc = fn
		return nil
	}
}

func (req *Request) debugf(format string, a ...interface{}) {
	if req.debugLogFunc != nil {
		req.debugLogFunc(logf(format, a...))
	}
}

func (req *Request) errorf(format string, a ...interface{}) {
	if req.errorLogFunc != nil {
		req.errorLogFunc(logf(format, a...))
	}
}

func logf(format string, a ...interface{}) string {
	return "fetcher: " + fmt.Sprintf(format, a...)
}
