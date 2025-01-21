package filter

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RequestContext stores the context between the different gRPC messages received from Envoy and it is used to store the request headers, response headers, and other information about the request.
// The Process method should be called on every message received from Envoy in order to update the request object.
// Note that the request object is not thread-safe and should not be shared between goroutines.
type RequestContext struct {
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	url             *url.URL
	cookies         []*http.Cookie
	status          int
	setCookies      []*http.Cookie
	metadata        *Metadata
	startTime       time.Time
}

// RequestHeader gets the first value associated with the given key.
// If there are no values associated with the key, RequestHeader returns "". It is case insensitive;
// [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key. RequestHeader assumes that all keys are stored in canonical form.
// To use non-canonical keys, access the map directly
func (r *RequestContext) RequestHeader(key string) string {
	if r.RequestHeaders == nil {
		return ""
	}
	return r.RequestHeaders.Get(key)
}

// RequestHeaderValues returns all values associated with the given key.
// It is case insensitive; [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key.
// To use non-canonical keys, access the map directly. The returned slice is not a copy.
func (r *RequestContext) RequestHeaderValues(key string) []string {
	if r.RequestHeaders == nil {
		return nil
	}
	return r.RequestHeaders.Values(key)
}

// ResponseHeader gets the first value associated with the given key.
// If there are no values associated with the key, ResponseHeader returns "". It is case insensitive;
// [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key. ResponseHeader assumes that all keys are stored in canonical form.
// To use non-canonical keys, access the map directly
func (r *RequestContext) ResponseHeader(key string) string {
	if r.ResponseHeaders == nil {
		return ""
	}
	return r.ResponseHeaders.Get(key)
}

// ResponseHeaderValues returns all values associated with the given key.
// It is case insensitive; [textproto.CanonicalMIMEHeaderKey] is used to canonicalize the provided key.
// To use non-canonical keys, access the map directly. The returned slice is not a copy.
func (r *RequestContext) ResponseHeaderValues(key string) []string {
	if r.ResponseHeaders == nil {
		return nil
	}
	return r.ResponseHeaders.Values(key)
}

// Scheme returns the scheme of the request (http or https)
func (r *RequestContext) Scheme() string {
	return r.RequestHeader(":scheme")
}

// Authority returns the authority of the request
func (r *RequestContext) Authority() string {
	return r.RequestHeader(":authority")
}

// Method returns the method of the request (GET, POST, PUT, etc)
func (r *RequestContext) Method() string {
	return r.RequestHeader(":method")
}

// URL returns the URL of the request
func (r *RequestContext) URL() *url.URL {
	if r.url != nil {
		return r.url
	}
	r.url, _ = url.Parse(r.RequestHeader(":path"))
	if r.url == nil {
		r.url = &url.URL{
			Path:    strings.Split(r.RequestHeader(":path"), "?")[0],
			RawPath: r.RequestHeader(":path"),
			User:    &url.Userinfo{},
		}
	}
	return r.url
}

// RequestID returns the request ID of the request
func (r *RequestContext) RequestID() string {
	return r.RequestHeader("x-request-id")
}

// Status returns the status of the response
func (r *RequestContext) Status() int {
	if r.status != 0 {
		return r.status
	}

	status, _ := strconv.Atoi(r.ResponseHeader(":status"))
	r.status = status
	return r.status
}

// StatusClass returns the class of the status of the response (2xx, 3xx, 4xx, 5xx)
func (r *RequestContext) StatusClass() string {
	return fmt.Sprintf("%dxx", r.Status()/100)
}

// Cookies returns a copy of the cookies of the request
func (r *RequestContext) Cookies() []http.Cookie {
	if r.cookies == nil && r.RequestHeader("cookie") != "" {
		httpreq := http.Request{Header: r.RequestHeaders}
		r.cookies = httpreq.Cookies()
	}

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
	if r.setCookies == nil && r.ResponseHeader("set-cookie") != "" {
		httpresp := http.Response{Header: r.ResponseHeaders}
		r.setCookies = httpresp.Cookies()
	}

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

// RequestDuration returns the time since the request started
func (r *RequestContext) RequestDuration() time.Duration {
	if r.startTime.IsZero() {
		r.startTime = time.Now()
	}
	return time.Since(r.startTime)
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

func NewRequestContext() *RequestContext {
	req := &RequestContext{
		RequestHeaders:  make(http.Header),
		ResponseHeaders: make(http.Header),
		startTime:       time.Now(),
	}

	return req
}
