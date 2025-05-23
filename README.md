# extproc-go

A Go library for building [Envoy External Processor](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_proc_filter).

> [!NOTE]
> This code is used in large scale production but is a work in progress and the API is subject to change.

## Example Usage

The following is an example of a filter that sets the SameSite attribute to Lax on all cookies set in the response headers. The filter is implemented as a Go struct that implements the `ResponseHeaders` method of the [filter.Filter](filter/filter.go) interface. The `ResponseHeaders` method is called by the extproc-go library when the filter is invoked by Envoy.

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

The filter is then registered with the extproc-go library and the server is started to listen for incoming requests from Envoy.

```go
package main

import (
	"context"

	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/server"
)

func main() {
	err := server.New(context.Background(),
		server.WithFilters(&filters.SameSiteLaxMode{}),
		server.WithEcho(),
	).Serve()

	// handle error ...
}
```

## Filter API

A server is composed of one of more filters. Requests and responses are proxied by Envoy to the external processor server,
and then each filter in turn receives request information. The current implementation supports implementing the following
methods:

- `RequestHeaders`: request headers is run on a request being made to the server
- `ResponseHeaders`: response headers is run on a response being returned from the downstream

The order of processing is determined by the order of filters passed to the server. On request, filters process the request
from first to last, while on response, filters process the request from last to first. This matches the [envoy implementation](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_filters#filter-ordering)
of filters.

## Stream API

Filters can also be run on changes to the stream by implementing the `Stream` interface. Currently, the stream interface
supports `OnStreamComplete`, which runs when a stream completes for any reason (e.g an `ImmediateResponse` is returned,
or extproc returns an `EOF`). `OnStreamComplete` allows adding a final async processing step, for instance emitting custom
metrics.

## Testing Filters

We provide a simple way to write and run integration tests against filters built with extproc-go. An example test case would look like the following:

```yaml
name: it should set set-cookie headers with SameSite=Lax and HttpOnly
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

The integration test requires [Envoy](examples/envoy.yml) and [extproc server](examples/main.go) running with the echo handlers loaded, the full setup is available in the [compose.yml](./examples/compose.yaml) file. To run the tests add the following to your test file:

```go
func TestSameSiteLax(t *testing.T) {
	// ...
	tc := extproctest.Load(t, "testdata/setcookie.yml")
	tc.Run(t)
}
```
