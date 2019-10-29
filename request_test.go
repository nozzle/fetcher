package fetcher

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
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
		client  *Client
		args    args
		want    *Request
		wantErr bool
	}{
		{
			"GET with headers",
			&Client{},
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "http://mywebsite.com",
				opts:   []RequestOption{WithAcceptJSONHeader()},
			},
			&Request{
				method:      "GET",
				url:         "http://mywebsite.com",
				maxAttempts: 1,
				headers: []header{
					{
						key:   "Accept",
						value: "application/json",
					},
				},
			},
			false,
		},
		{
			"client parent options - GET with headers",
			&Client{parentRequestOptions: []RequestOption{WithAcceptJSONHeader()}},
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "http://mywebsite.com",
				opts:   []RequestOption{},
			},
			&Request{
				method:      "GET",
				url:         "http://mywebsite.com",
				maxAttempts: 1,
				headers: []header{
					{
						key:   "Accept",
						value: "application/json",
					},
				},
			},
			false,
		},
		{
			"client with parent options - POST with URLEncoded payload",
			&Client{parentRequestOptions: []RequestOption{WithAcceptJSONHeader()}},
			args{
				c:      ctx,
				method: http.MethodPost,
				url:    "http://mywebsite.com",
				opts: []RequestOption{WithURLEncodedPayload(url.Values(map[string][]string{
					"a": []string{"1"},
					"b": []string{"2"},
					"c": []string{"3"},
				}))},
			},
			&Request{
				method:      "POST",
				url:         "http://mywebsite.com",
				maxAttempts: 1,
				headers: []header{
					{
						key:   "Accept",
						value: "application/json",
					},
					{
						key:   "Content-Type",
						value: "application/x-www-form-urlencoded",
					},
				},
				payload: bytes.NewBufferString("a=1&b=2&c=3"),
			},
			false,
		},
		{
			"GET with params including parent options - params are sorted by key",
			&Client{parentRequestOptions: []RequestOption{
				WithParam("fizzle", "dizzle"),
			}},
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "http://mywebsite.com",
				opts: []RequestOption{
					WithParam("foo", "bar"),
					WithParam("dan: %shay", "c# programming"),
				},
			},
			&Request{
				method:      "GET",
				url:         "http://mywebsite.com?dan%3A+%25shay=c%23+programming&fizzle=dizzle&foo=bar",
				maxAttempts: 1,
			},
			false,
		},
		{
			"erroring option - GET",
			&Client{},
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
			"WithBaseURL RequestOption",
			&Client{parentRequestOptions: []RequestOption{
				WithBaseURL("https://mywebsite.com"),
				WithParam("foo", "bar"),
			}},
			args{
				c:      ctx,
				method: http.MethodGet,
				url:    "/blog",
				opts:   []RequestOption{},
			},
			&Request{
				method:      "GET",
				url:         "https://mywebsite.com/blog?foo=bar",
				maxAttempts: 1,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client.NewRequest(tt.args.c, tt.args.method, tt.args.url, tt.args.opts...)
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
