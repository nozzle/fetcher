# fetcher [![CircleCI](https://img.shields.io/circleci/project/github/nozzle/fetcher.svg)](https://circleci.com/gh/nozzle/fetcher) [![GoDoc](https://godoc.org/github.com/nozzle/fetcher?status.svg)](https://godoc.org/github.com/nozzle/fetcher) [![Codecov](https://img.shields.io/codecov/c/github/nozzle/fetcher.svg)](https://codecov.io/gh/nozzle/fetcher/) [![Go Report Card](https://goreportcard.com/badge/github.com/nozzle/fetcher)](https://goreportcard.com/report/github.com/nozzle/fetcher)
HTTP Client - Simplified - Mockable

1. Create your client
- A Client is required to make any http calls - good thing it's dead easy to create:
- Simple Client example:

```
  c := context.Background()
  cl, err := github.NewClient(c)
```
- Advanced Client example:
```
  c := context.Background()
  cl, err := fetcher.NewClient(c,
    fetcher.WithRequestOptions([]fetcher.RequestOption{
      fetcher.WithAcceptJSONHeader(),
      fetcher.WithHeader("API-Token", os.Getenv("API_TOKEN")),
    }),
  )
```
- This client can now be used as much as needed. 

2. Pass your client to the function as a fetcher.Fetcher interface object:
```
  func sharedCount(c context.Context, f fetcher.Fetcher, uri string) (int, error) {
	
    ...

    return countResp.Count, nil
  }

```
- This function is now testable with a mocked client using fetchermock.
  * See [fetchermock/client_test.go:TestSharedCount](https://github.com/nozzle/fetcher/blob/807c82bdfff749ca61c7cd3e80364fd119d42533/fetchermock/client_test.go#L22)

3. Use your client to make a call:
```
func sharedCount(c context.Context, f fetcher.Fetcher, uri string) (int, error) {
  apiURL := "http://www.linkedin.com/countserv/count/share?format=json&url=" + url.QueryEscape(uri)
  resp, err := f.Get(c, apiURL, fetcher.WithMaxAttempts(3))
  if err != nil {
    return 0, err
  }
```

4. Handle the response
```
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

  return countResp.Count, nil`
}

```

5. Write your test
- you can now use `fetchermock` to create a testing client that satisfies the fetcher.Fetcher interface - including the expected response body, status code, and/or error. 

Advanced features:
1. Retry loop
2. Copied response body for easier debugging
3. Rate Limiting
4. Max Idle Connections Per Host
5. Custom Debug/Error Logging
6. Request Backoff Options
