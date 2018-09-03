package fetchermock_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/nozzle/fetcher"
	"github.com/nozzle/fetcher/fetchermock"
)

func TestSharedCount(t *testing.T) {
	type args struct {
		c          context.Context
		uri        string
		reqURL     string
		statusCode int
		respBody   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			"Standard implementation",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://www.linkedin.com/countserv/count/share?format=json&url=https%3A%2F%2Fnozzle.io",
				200,
				[]byte(`{"count":29,"fCnt":"29","fCntPlusOne":"30","url":"https:\/\/nozzle.io\/"}`),
			},
			29,
			false,
		},
		{
			"Bad JSON",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://www.linkedin.com/countserv/count/share?format=json&url=https%3A%2F%2Fnozzle.io",
				200,
				[]byte(`"count":29,"fCnt":"29","fCntPlusOne":"30","url":"https:\/\/nozzle.io\/"`),
			},
			0,
			true,
		},
		{
			"invalid url",
			args{
				context.Background(),
				"www.cedartreeinsurance.com",
				"http://www.linkedin.com/countserv/count/share?format=json&url=www.cedartreeinsurance.com",
				400,
				[]byte(`Invalid URL parameter: www.cedartreeinsurance.com`),
			},
			0,
			true,
		},
		{
			"bad status code",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://www.linkedin.com/countserv/count/share?format=json&url=https%3A%2F%2Fnozzle.io",
				500,
				[]byte(`{}`),
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
				fetchermock.WithResponseStatusCode(tt.args.statusCode),
				fetchermock.WithResponseBodyBytes(tt.args.respBody),
				fetchermock.WithResponseHeader(fetcher.ContentTypeHeader, fetcher.ContentTypeJSON),
			)
			// *** END FETCHERMOCK SETUP ***

			count, err := sharedCount(tt.args.c, fm, tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("sharedCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				fmt.Println(err.Error())
				return
			}

			if count != tt.want {
				t.Errorf("sharedCount() = %v, want %v", count, tt.want)
			}
		})
	}
}

// sharedCount returns the Linkedin Shared Count as obtained from their api
func sharedCount(c context.Context, f fetcher.Fetcher, uri string) (int, error) {
	apiURL := "http://www.linkedin.com/countserv/count/share?format=json&url=" + url.QueryEscape(uri)
	resp, err := f.Get(c, apiURL, fetcher.WithMaxAttempts(3))
	if err != nil {
		return 0, err
	}
	defer resp.Close()

	switch {
	case resp.StatusCode() == 400:
		return 0, errors.New("invalid url")
	case resp.StatusCode() > 300:
		return 0, errors.New("bad status code")
	}

	type countResponse struct {
		Count int    `json:"count"`
		URL   string `json:"url"`
	}

	countResp := &countResponse{}
	if err = resp.Decode(c, countResp, fetcher.WithJSONBody()); err != nil {
		return 0, err
	}

	return countResp.Count, nil
}
