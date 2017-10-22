package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/nozzle/nozzle/pkg/tester"
)

func TestNewClient(t *testing.T) {
	var tests = []struct {
		Desc                    string
		KeepAlive               time.Duration
		HandshakeTimeout        time.Duration
		ExpKeepAlive            time.Duration
		ExpKeepHandshakeTimeout time.Duration
		ExpErr                  string
	}{
		{
			"Standard implementation",
			15 * time.Second,
			30 * time.Second,
			15 * time.Second,
			30 * time.Second,
			"",
		},
	}

	for _, test := range tests {
		c := context.Background()
		cl, err := NewClient(c,
			ClientWithKeepAlive(test.KeepAlive),
			ClientWithHandshakeTimeout(test.HandshakeTimeout),
		)

		tester.Equal(t, test, test.ExpKeepAlive, cl.keepAlive, test.Desc)
		tester.Error(t, test, test.ExpErr, err, test.Desc)
	}
}
