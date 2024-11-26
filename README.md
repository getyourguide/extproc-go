# extproc-go

A Go framework for building [Envoy External Processor](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_proc_filter).

> [!NOTE]
> This is a work in progress and the API is subject to change.

## Example Usage

- [SameSiteLaxMode](./examples/filters/setcookie.go)

```go
type SameSiteLaxMode struct {
	filter.NoOpFilter
}

func (f *SameSiteLaxMode) ResponseHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	for i, cookie := range req.SetCookies() {
		cookie.SameSite = http.SameSiteLaxMode
		cookie.HttpOnly = true
		// The first set-cookie we overwrite the header, the others we append
		if i == 0 {
			crw.SetHeader("set-cookie", cookie.String())
			continue
		}
		crw.AppendHeader("set-cookie", cookie.String())
	}
	return nil, nil
}
```

## Testing Filters

We provide a simple way to write and run integration tests against filters built with extproc-go. An example test case would look like the following:

```yaml
name: it should match on match type ANY
input:
  headers:
    - name: path
      value: /response-headers?set-cookie=session=my-session&set-cookie=auth=d2h5IHNvIGN1cmlvdXM/Cg==
expect:
  responseHeaders:
    - name: set-cookie
      exact: session=my-session; HttpOnly; SameSite=Lax
      matchAction: ANY
    - name: set-cookie
      exact: auth=d2h5IHNvIGN1cmlvdXM/Cg==; HttpOnly; SameSite=Lax
      matchAction: ANY
```

The integration test requires [Envoy](examples/envoy.yml) and [extproc server](examples/main.go) running with the echo handlers loaded, the full setup is available in the [docker-compose.yml](./docker-compose.yml) file. To run the tests add the following to your test file:

```go
func TestSameSiteLax(t *testing.T) {
	tc := httptest.Load(t, "testdata/setcookie.yml")
	tc.Run(t)
}
```
