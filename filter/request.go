package filter

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

// RequestPhase represents the different phases of the request
type RequestPhase string

const (
	RequestPhaseUnknown          RequestPhase = "RequestPhaseUnknown"
	RequestPhaseRequestHeaders   RequestPhase = "RequestPhaseRequestHeaders"
	RequestPhaseRequestBody      RequestPhase = "RequestPhaseRequestBody"
	RequestPhaseRequestTrailers  RequestPhase = "RequestPhaseRequestTrailers"
	RequestPhaseResponseHeaders  RequestPhase = "RequestPhaseResponseHeaders"
	RequestPhaseResponseBody     RequestPhase = "RequestPhaseResponseBody"
	RequestPhaseResponseTrailers RequestPhase = "RequestPhaseResponseTrailers"
)

// RequestContext stores the context between the different gRPC messages received from Envoy and it is used to store the request headers, response headers, and other information about the request.
// The Process method should be called on every message received from Envoy in order to update the request object.
// Note that the request object is not thread-safe and should not be shared between goroutines.
type RequestContext struct {
	scheme         string
	authority      string
	method         string
	url            *url.URL
	requestID      string
	status         int
	mutatedHeaders map[RequestPhase]http.Header
	rawHeaders     map[RequestPhase]http.Header
	cookies        []*http.Cookie
	setCookies     []*http.Cookie
	metadata       *Metadata
	phase          RequestPhase
	startTime      time.Time
}

// RequestHeader gets the first value associated with the given key.
// If there are no values associated with the key, RequestHeader returns "". It is case insensitive;
// [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key. RequestHeader assumes that all keys are stored in canonical form.
// To use non-canonical keys, access the map directly
func (r *RequestContext) RequestHeader(key string) string {
	if r.rawHeaders[RequestPhaseRequestHeaders] == nil {
		return ""
	}
	return r.rawHeaders[RequestPhaseRequestHeaders].Get(key)
}

// RequestHeaderValues returns all values associated with the given key.
// It is case insensitive; [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key.
// To use non-canonical keys, access the map directly. The returned slice is not a copy.
func (r *RequestContext) RequestHeaderValues(key string) []string {
	if r.rawHeaders[RequestPhaseRequestHeaders] == nil {
		return nil
	}
	return r.rawHeaders[RequestPhaseRequestHeaders].Values(key)
}

// ResponseHeader gets the first value associated with the given key.
// If there are no values associated with the key, ResponseHeader returns "". It is case insensitive;
// [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key. ResponseHeader assumes that all keys are stored in canonical form.
// To use non-canonical keys, access the map directly
func (r *RequestContext) ResponseHeader(key string) string {
	if r.rawHeaders[RequestPhaseResponseHeaders] == nil {
		return ""
	}
	return r.rawHeaders[RequestPhaseResponseHeaders].Get(key)
}

// ResponseHeaderValues returns all values associated with the given key.
// It is case insensitive; [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key.
// To use non-canonical keys, access the map directly. The returned slice is not a copy.
func (r *RequestContext) ResponseHeaderValues(key string) []string {
	if r.rawHeaders[RequestPhaseResponseHeaders] == nil {
		return nil
	}
	return r.rawHeaders[RequestPhaseResponseHeaders].Values(key)
}

// MutatedHeaders returns a copy of headers mutated in the given phase.
func (r *RequestContext) MutatedHeaders(phase RequestPhase) http.Header {
	if r.mutatedHeaders[phase] == nil {
		r.mutatedHeaders[phase] = make(http.Header)
	}
	return r.mutatedHeaders[phase].Clone()
}

// RawHeaders returns a copy of the raw headers from the given phase.
func (r *RequestContext) RawHeaders(phase RequestPhase) http.Header {
	if r.rawHeaders[phase] == nil {
		r.rawHeaders[phase] = make(http.Header)
	}
	return r.rawHeaders[phase].Clone()
}

// Scheme returns the scheme of the request (http or https)
func (r *RequestContext) Scheme() string {
	return r.scheme
}

// Authority returns the authority of the request
func (r *RequestContext) Authority() string {
	return r.authority
}

// Method returns the method of the request (GET, POST, PUT, etc)
func (r *RequestContext) Method() string {
	return r.method
}

// URL returns the URL of the request
func (r *RequestContext) URL() *url.URL {
	if r.url == nil {
		return &url.URL{
			User: &url.Userinfo{},
		}
	}
	return r.url
}

// RequestID returns the request ID of the request
func (r *RequestContext) RequestID() string {
	return r.requestID
}

// Status returns the status of the response
func (r *RequestContext) Status() int {
	return r.status
}

// StatusClass returns the class of the status of the response (2xx, 3xx, 4xx, 5xx)
func (r *RequestContext) StatusClass() string {
	return fmt.Sprintf("%dxx", r.status/100)
}

// Cookies returns a copy of the cookies of the request
func (r *RequestContext) Cookies() []http.Cookie {
	cookies := make([]http.Cookie, len(r.cookies))
	for i, c := range r.cookies {
		cookies[i] = *c
	}
	return cookies
}

// GetCookie returns the cookie with the given name and a boolean indicating if the cookie was found
func (r *RequestContext) GetCookie(name string) (http.Cookie, bool) {
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return http.Cookie{}, false
}

// SetCookies returns a copy of the cookies from set-cookies response headers
func (r *RequestContext) SetCookies() []http.Cookie {
	cookies := make([]http.Cookie, len(r.setCookies))
	for i, c := range r.setCookies {
		cookies[i] = *c
	}
	return cookies
}

// Metadata returns the metadata of the request, it can be used to excange information between the different filters
func (r *RequestContext) Metadata() *Metadata {
	if r.metadata == nil {
		r.metadata = &Metadata{}
	}
	return r.metadata
}

// RequestPhase returns the current phase of the request
func (r *RequestContext) RequestPhase() RequestPhase {
	return r.phase
}

// RequestDuration returns the time since the request started
func (r *RequestContext) RequestDuration() time.Duration {
	if r.startTime.IsZero() {
		return time.Duration(0)
	}
	return time.Since(r.startTime)
}

// Process processes the given message and updates the request object accordingly
// It should be called on every message received from Envoy
func (r *RequestContext) Process(message any, mutatedHeaders http.Header) {
	if r.mutatedHeaders == nil {
		r.mutatedHeaders = make(map[RequestPhase]http.Header)
	}
	if r.rawHeaders == nil {
		r.rawHeaders = make(map[RequestPhase]http.Header)
	}
	if r.metadata == nil {
		r.metadata = &Metadata{}
	}
	if r.startTime.IsZero() {
		r.startTime = time.Now()
	}
	switch msg := any(message).(type) {
	case *extproc.ProcessingRequest_RequestHeaders:
		r.phase = RequestPhaseRequestHeaders
		r.mutatedHeaders[RequestPhaseRequestHeaders] = mutatedHeaders
		r.rawHeaders[RequestPhaseRequestHeaders] = make(http.Header)

		for _, header := range msg.RequestHeaders.GetHeaders().GetHeaders() {
			headerValue := cmp.Or(string(header.GetRawValue()), header.GetValue())
			r.rawHeaders[RequestPhaseRequestHeaders].Add(header.Key, headerValue)
			r.mutatedHeaders[RequestPhaseRequestHeaders].Add(header.Key, headerValue)
		}
		if r.scheme == "" {
			r.scheme = r.RequestHeader(":scheme")
		}
		if r.authority == "" {
			r.authority = r.RequestHeader(":authority")
		}
		if r.method == "" {
			r.method = r.RequestHeader(":method")
		}
		if r.requestID == "" {
			r.requestID = r.RequestHeader("x-request-id")
		}
		if r.url == nil {
			r.url, _ = url.Parse(r.RequestHeader(":path"))
			if r.url == nil {
				r.url = &url.URL{
					Path:    strings.Split(r.RequestHeader(":path"), "?")[0],
					RawPath: r.RequestHeader(":path"),
					User:    &url.Userinfo{},
				}
			}
		}
		if r.cookies == nil && r.RequestHeader("cookie") != "" {
			httpreq := http.Request{Header: r.rawHeaders[RequestPhaseRequestHeaders]}
			r.cookies = httpreq.Cookies()
		}

	case *extproc.ProcessingRequest_RequestBody:
		// Not implemented
		r.phase = RequestPhaseRequestBody
	case *extproc.ProcessingRequest_RequestTrailers:
		// Not implemented
		r.phase = RequestPhaseRequestTrailers
	case *extproc.ProcessingRequest_ResponseTrailers:
		// Not implemented
		r.phase = RequestPhaseResponseTrailers
	case *extproc.ProcessingRequest_ResponseBody:
		// Not implemented
		r.phase = RequestPhaseResponseBody
	case *extproc.ProcessingRequest_ResponseHeaders:
		r.phase = RequestPhaseResponseHeaders
		r.mutatedHeaders[RequestPhaseResponseHeaders] = mutatedHeaders
		r.rawHeaders[RequestPhaseResponseHeaders] = make(http.Header)

		for _, header := range msg.ResponseHeaders.GetHeaders().GetHeaders() {
			headerValue := cmp.Or(string(header.GetRawValue()), header.GetValue())
			r.rawHeaders[RequestPhaseResponseHeaders].Add(header.Key, headerValue)
			r.mutatedHeaders[RequestPhaseResponseHeaders].Add(header.Key, headerValue)
		}

		status, _ := strconv.Atoi(r.ResponseHeader(":status"))
		r.status = status

		if r.setCookies == nil && r.ResponseHeader("set-cookie") != "" {
			httpresp := http.Response{Header: r.rawHeaders[RequestPhaseResponseHeaders]}
			r.setCookies = httpresp.Cookies()
		}
	case *extproc.ProcessingResponse_ResponseTrailers:
		r.phase = RequestPhaseResponseTrailers
	}
}

type Metadata struct {
	m map[any]any
}

// Set sets the value associated with key in the metadata.
func (m *Metadata) Set(key any, value any) {
	if m.m == nil {
		m.m = make(map[any]any)
	}
	m.m[key] = value
}

// Get returns the value associated with key in the metadata.
func (m *Metadata) Get(key any) any {
	if m.m == nil {
		m.m = make(map[any]any)
	}
	return m.m[key]
}
