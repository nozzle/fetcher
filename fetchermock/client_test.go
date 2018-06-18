package fetchermock_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/nozzle/fetcher"
	"github.com/nozzle/fetcher/fetchermock"
)

func TestSharedCount(t *testing.T) {
	type args struct {
		c        context.Context
		uri      string
		reqURL   string
		respBody []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			"Good_JSON",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://api.pinterest.com/v1/urls/count.json?callback=receiveCount&url=https%3A%2F%2Fnozzle.io",
				[]byte(`{"url":"https://nozzle.io/","count":30}`),
			},
			30,
			false,
		},
		{
			"Bad_JSON",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://api.pinterest.com/v1/urls/count.json?callback=receiveCount&url=https%3A%2F%2Fnozzle.io",
				[]byte(`({"url":"https://nozzle.io/","count":30}`),
			},
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// *** BEGIN FETCHERMOCK SETUP ***
			fm, err := fetchermock.NewClient(tt.args.c, fetchermock.WithExpectationsInOrder(true))
			if err != nil {
				t.Fatal(err)
			}
			fm.ExpectRequest(tt.args.c, http.MethodGet, tt.args.reqURL,
				fetchermock.WithRequestOptions(
					fetcher.WithMaxAttempts(3),
				),
				fetchermock.WithResponseStatusCode(200),
				fetchermock.WithResponseBodyBytes(tt.args.respBody),
				fetchermock.WithResponseHeader(fetcher.ContentTypeHeader, fetcher.ContentTypeJSON),
			)
			// *** END FETCHERMOCK SETUP ***

			got, err := sharedCount(tt.args.c, fm, tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("sharedCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("sharedCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func sharedCount(c context.Context, f fetcher.Fetcher, uri string) (int, error) {
	apiURL := "http://api.pinterest.com/v1/urls/count.json?callback=receiveCount&url=" + url.QueryEscape(uri)
	resp, err := f.Get(c, apiURL,
		fetcher.WithMaxAttempts(3),
		fetcher.WithAfterDoFunc(func(req *fetcher.Request, resp *fetcher.Response) error {
			if resp.StatusCode() >= 500 {
				return errors.New("Status Code Error")
			}
			return nil
		}),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Close()

	type countResponse struct {
		URL   string
		Count int
	}

	countResp := &countResponse{}
	if err = resp.Decode(c, countResp, fetcher.WithJSONBody()); err != nil {
		return 0, err
	}

	return countResp.Count, nil
}
