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

type args struct {
	c           context.Context
	uri         string
	reqURL      string
	respBody    []byte
	requestBody []byte
}

func TestSharedCount(t *testing.T) {
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
				nil,
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
				nil,
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
				fetchermock.WithResponseStatusCode(http.StatusOK),
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
			if resp.StatusCode() >= http.StatusInternalServerError {
				return errors.New("status code error")
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

func TestClient_Post(t *testing.T) {
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"Populated Request Body",
			args{
				context.Background(),
				"https://nozzle.io",
				"https://nozzle.io/api/test",
				[]byte(`{"url":"https://nozzle.io/","count":30}`),
				[]byte(`{"url": "https://nozzle.io"}`),
			},
			[]byte(`{"url": "https://nozzle.io"}`),
			false,
		},
		{
			"Empty Request Body",
			args{
				context.Background(),
				"https://nozzle.io",
				"http://api.pinterest.com/v1/urls/count.json?callback=receiveCount&url=https%3A%2F%2Fnozzle.io",
				[]byte(`({"url":"https://nozzle.io/","count":30}`),
				[]byte("hello world"),
			},
			[]byte(""),
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

			err = fm.ExpectRequest(tt.args.c, http.MethodPost, tt.args.reqURL,
				fetchermock.WithRequestOptions(
					fetcher.WithMaxAttempts(3),
				),
				fetchermock.WithRequestOptions(fetcher.WithBytesPayload(tt.want)),
				fetchermock.WithResponseStatusCode(http.StatusOK),
				fetchermock.WithResponseBodyBytes(tt.args.respBody),
				fetchermock.WithResponseHeader(fetcher.ContentTypeHeader, fetcher.ContentTypeJSON),
			)
			if err != nil {
				t.Fatal(err)
			}
			// *** END FETCHERMOCK SETUP ***

			_, err = fm.Post(tt.args.c, tt.args.reqURL, fetcher.WithBytesPayload(tt.args.requestBody))
			if (err != nil) != tt.wantErr {
				t.Fatalf("got %v error wanted %v", err, tt.wantErr)
			}
		})
	}

}
