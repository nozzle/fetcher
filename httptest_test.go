package fetcher

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type serverData struct {
	headers       map[string]string
	body          []byte
	encodableData interface{}
	statusCode    int
}

type testObject struct {
	URL   string
	Count int
}

func testLogFunc(t *testing.T) LogFunc {
	return func(s string) {
		t.Log(s)
	}
}

func testServerHelper(t *testing.T, sd *serverData) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if there's no explicit body and an interface to marshal, set the body
		if sd.body == nil && sd.encodableData != nil {
			var err error
			// encode if an appropriate Accept header is supplied
			switch r.Header.Get(AcceptHeader) {
			case ContentTypeJSON:
				sd.body, err = json.Marshal(sd.encodableData)
				if err != nil {
					t.Errorf("encoding failed in test server: %v", err)
					w.WriteHeader(9999)
					return
				}

			case ContentTypeGob:
				buf := getBuffer()
				defer putBuffer(buf)
				err = gob.NewEncoder(buf).Encode(sd.encodableData)
				if err != nil {
					t.Errorf("encoding failed in test server: %v", err)
					w.WriteHeader(9999)
					return
				}
				sd.body = buf.Bytes()

			case ContentTypeXML:
				sd.body, err = xml.Marshal(sd.encodableData)
				if err != nil {
					t.Errorf("encoding failed in test server: %v", err)
					w.WriteHeader(9999)
					return
				}
			}
		}

		for k, v := range sd.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(sd.statusCode)
		w.Write(sd.body)
	}))
}

func TestEndToEndWithObject(t *testing.T) {
	tests := []struct {
		name           string
		c              context.Context
		clientOptions  []ClientOption
		method         string
		requestOptions []RequestOption
		serverData     *serverData
		decodeOptions  []DecodeOption
		want           testObject
	}{
		{
			"Basic JSON",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{},
			&serverData{
				headers:    map[string]string{ContentTypeHeader: ContentTypeJSON},
				body:       []byte(`{"URL":"https://nozzle.io/","Count":30}`),
				statusCode: 200,
			},
			[]DecodeOption{DecodeWithJSON()},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic JSON detect encoding",
			context.Background(),
			[]ClientOption{ClientWithDebugLogFunc(testLogFunc(t))},
			http.MethodGet,
			[]RequestOption{RequestWithAcceptJSONHeader()},
			&serverData{
				headers:       map[string]string{ContentTypeHeader: ContentTypeJSON},
				encodableData: testObject{URL: "https://nozzle.io/", Count: 30},
				statusCode:    200,
			},
			[]DecodeOption{},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic JSON with custom decode func",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{},
			&serverData{
				headers:    map[string]string{ContentTypeHeader: ContentTypeJSON},
				body:       []byte(`{"URL":"https://nozzle.io/","Count":30}`),
				statusCode: 200,
			},
			[]DecodeOption{DecodeWithCustomFunc(jsonDecodeFunc)},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic Gob",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{RequestWithHeader(AcceptHeader, ContentTypeGob)},
			&serverData{
				headers:       map[string]string{ContentTypeHeader: ContentTypeGob},
				encodableData: testObject{URL: "https://nozzle.io/", Count: 30},
				statusCode:    200,
			},
			[]DecodeOption{DecodeWithGob()},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic Gob detect encoding",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{RequestWithHeader(AcceptHeader, ContentTypeGob)},
			&serverData{
				headers:       map[string]string{ContentTypeHeader: ContentTypeGob},
				encodableData: testObject{URL: "https://nozzle.io/", Count: 30},
				statusCode:    200,
			},
			[]DecodeOption{},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic XML",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{RequestWithHeader(AcceptHeader, ContentTypeXML)},
			&serverData{
				headers:       map[string]string{ContentTypeHeader: ContentTypeXML},
				encodableData: testObject{URL: "https://nozzle.io/", Count: 30},
				statusCode:    200,
			},
			[]DecodeOption{DecodeWithXML()},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
		{
			"Basic XML detect encoding",
			context.Background(),
			[]ClientOption{},
			http.MethodGet,
			[]RequestOption{RequestWithHeader(AcceptHeader, ContentTypeXML)},
			&serverData{
				headers:       map[string]string{ContentTypeHeader: ContentTypeXML},
				encodableData: testObject{URL: "https://nozzle.io/", Count: 30},
				statusCode:    200,
			},
			[]DecodeOption{},
			testObject{URL: "https://nozzle.io/", Count: 30},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := testServerHelper(t, tt.serverData)
			defer ts.Close()

			cl, err := NewClient(tt.c, tt.clientOptions...)
			if err != nil {
				t.Errorf("NewClient failed: %v", err)
				return
			}

			req, err := NewRequest(tt.c, tt.method, ts.URL, tt.requestOptions...)
			if err != nil {
				t.Errorf("NewRequest failed: %v", err)
				return
			}

			resp, err := cl.Do(tt.c, req)
			if err != nil {
				t.Errorf("cl.Do failed: %v", err)
				return
			}

			got := testObject{}
			err = resp.Decode(tt.c, &got, tt.decodeOptions...)
			if err != nil {
				t.Errorf("resp.Decode failed: %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
