package fetcher

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	type args struct {
		c    context.Context
		opts []ClientOption
	}
	tests := []struct {
		name    string
		args    args
		want    *Client
		wantErr bool
	}{
		{
			"Standard implementation",
			args{
				context.Background(),
				[]ClientOption{
					WithKeepAlive(15 * time.Second),
					WithHandshakeTimeout(30 * time.Second),
					WithMaxIdleConnsPerHost(20),
				},
			},
			&Client{
				keepAlive:           15 * time.Second,
				handshakeTimeout:    30 * time.Second,
				maxIdleConnsPerHost: 20,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.c, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got.client = nil // not comparing the *http.Client, just the *Client
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
