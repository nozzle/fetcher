package fetcher

import "context"
import "fmt"

// LogFunc is a pluggable log function
type LogFunc func(string)

// ClientWithDebugLogFunc pipes all debug logs to the supplied function
// All requests from this client inherit this logger
func ClientWithDebugLogFunc(fn LogFunc) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.debugLogFunc = fn
		return nil
	}
}

// ClientWithErrorLogFunc pipes all error logs to the supplied function
// All requests from this client inherit this logger
func ClientWithErrorLogFunc(fn LogFunc) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.errorLogFunc = fn
		return nil
	}
}

// RequestWithDebugLogFunc pipes all debug logs to the supplied function
// This overrides and replaces the inherited client functions
func RequestWithDebugLogFunc(fn LogFunc) RequestOption {
	return func(c context.Context, req *Request) error {
		req.debugLogFunc = fn
		return nil
	}
}

// RequestWithErrorLogFunc pipes all error logs to the supplied function
// This overrides and replaces the inherited client functions
func RequestWithErrorLogFunc(fn LogFunc) RequestOption {
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
