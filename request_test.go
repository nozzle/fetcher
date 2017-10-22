package fetcher

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestNewRequest(t *testing.T) {
	ctx := context.Background()
	type args struct {
		c      context.Context
		method string
		url    string
		opts   []RequestOption
	}
	tests := []struct {
		name    string
		args    args
		want    *Request
		wantErr bool
	}{
		{
			"no options - GET with headers",
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "http://mywebsite.com",
				opts:   []RequestOption{RequestWithAcceptJSONHeader()},
			},
			&Request{
				method:      "GET",
				url:         "http://mywebsite.com",
				maxAttempts: 1,
				headers:     map[string]string{"Accept": "application/json"},
			},
			false,
		},
		{
			"erroring option - GET",
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "http://mywebsite.com",
				opts:   []RequestOption{func(c context.Context, req *Request) error { return errors.New("test error") }},
			},
			nil,
			true,
		},
		{
			"http.NewRequest url parse error",
			args{
				c:      ctx,
				method: "BADMETHOD",
				url:    "http:/www.mywebsite.com",
				opts:   []RequestOption{},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRequest(tt.args.c, tt.args.method, tt.args.url, tt.args.opts...)
			switch {
			case tt.wantErr && err != nil:
				return
			case tt.wantErr && err == nil:
				t.Fatalf("NewRequest() error = nil, wantErr %t", tt.wantErr)
			case !tt.wantErr && err != nil:
				t.Fatalf("NewRequest() error = %v, wantErr %t", err, tt.wantErr)
			}

			if equal, info := tt.want.Equal(got); !equal {
				t.Errorf("NewRequest() = %s, want %s", got.String(), tt.want.String())
				t.Errorf("info: %s", info)
			}
		})
	}
}
