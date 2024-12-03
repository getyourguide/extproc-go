package filter_test

import (
	"net/http"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/stretchr/testify/require"
)

const (
	requestID = "30149a57-c842-9e40-968d-bf9bdbed55b1"
)

var (
	envoyHeadersValue = []*corev3.HeaderValue{
		{Key: ":scheme", Value: "https"},
		{Key: ":authority", Value: "example.com"},
		{Key: ":method", Value: "GET"},
		{Key: ":path", Value: "/?q=a"},
		{Key: "x-request-id", Value: requestID},
	}

	envoyHeadersRawValue = []*corev3.HeaderValue{
		{Key: ":scheme", RawValue: []byte("https")},
		{Key: ":authority", RawValue: []byte("example.com")},
		{Key: ":method", RawValue: []byte("GET")},
		{Key: ":path", RawValue: []byte("/?q=a")},
		{Key: "x-request-id", RawValue: []byte(requestID)},
	}
)

func TestMutatedRequestHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers []*corev3.HeaderValue
		assert  func(t *testing.T, req *filter.RequestContext)
		mutate  func(t *testing.T, crw *filter.CommonResponseWriter)
	}{{
		name:    "set headers",
		headers: envoyHeadersRawValue,
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.SetHeader("header-a", "value-a")
			crw.SetHeader("header-b", "value-b")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			mutatedHeaders := req.MutatedHeaders(filter.RequestPhaseRequestHeaders)
			require.Equal(t, "value-a", mutatedHeaders.Get("header-a"))
			require.Equal(t, "value-b", mutatedHeaders.Get("header-b"))
		},
	}, {
		name: "append headers",
		headers: func() []*corev3.HeaderValue {
			headers := envoyHeadersRawValue
			headers = append(headers, &corev3.HeaderValue{Key: "header-c", RawValue: []byte("value-c")})
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.AppendHeader("header-c", "value-c1")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			mutatedHeaders := req.MutatedHeaders(filter.RequestPhaseRequestHeaders)
			got := mutatedHeaders.Values("header-c")
			want := []string{"value-c", "value-c1"}
			require.Equal(t, want, got)
		},
	}, {
		name: "remove headers",
		headers: func() []*corev3.HeaderValue {
			headers := envoyHeadersRawValue
			headers = append(headers, &corev3.HeaderValue{Key: "header-c", RawValue: []byte("value-c")})
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.RemoveHeaders("header-c")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			mutatedHeaders := req.MutatedHeaders(filter.RequestPhaseRequestHeaders)
			got := mutatedHeaders.Get("header-c")
			want := ""
			require.Equal(t, want, got)
		},
	}, {
		name: "cross message access",
		headers: func() []*corev3.HeaderValue {
			headers := envoyHeadersRawValue
			headers = append(headers, &corev3.HeaderValue{Key: "header-c", RawValue: []byte("value-c")})
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.SetHeader("header-d", "value-d")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			respMsg := &extproc.ProcessingRequest_ResponseHeaders{
				ResponseHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{},
				},
			}
			headers := make(http.Header)
			req.Process(respMsg, headers)
			mutatedHeaders := req.MutatedHeaders(filter.RequestPhaseRequestHeaders)
			require.Equal(t, "value-d", mutatedHeaders.Get("header-d"))
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			msg := &extproc.ProcessingRequest_RequestHeaders{
				RequestHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: tt.headers,
					},
				},
			}
			req := &filter.RequestContext{}
			headers := make(http.Header)
			req.Process(msg, headers)

			crw := filter.NewCommonResponseWriter(headers)
			tt.mutate(t, crw)

			tt.assert(t, req)
		})
	}
}

func TestMetadata(t *testing.T) {
	msg := &extproc.ProcessingRequest_RequestHeaders{
		RequestHeaders: &extproc.HttpHeaders{
			Headers: &corev3.HeaderMap{
				Headers: envoyHeadersValue,
			},
		},
	}
	t.Run("set and get", func(t *testing.T) {
		req := filter.RequestContext{}
		headers := make(http.Header)
		req.Process(msg, headers)

		req.Metadata().Set("key", "value")
		require.Equal(t, "value", req.Metadata().Get("key"))
	})

	t.Run("get non-existent key", func(t *testing.T) {
		req := filter.RequestContext{}
		headers := make(http.Header)
		req.Process(msg, headers)

		require.Empty(t, req.Metadata().Get("non-existent"))
	})
}

func TestProcessRequestHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers []*corev3.HeaderValue
		assert  func(t *testing.T, req *filter.RequestContext)
	}{{
		name:    "empty headers",
		headers: []*corev3.HeaderValue{},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, req.RequestPhase(), filter.RequestPhaseRequestHeaders)
			require.Empty(t, req.RequestID())
			require.Empty(t, req.Authority())
			require.Empty(t, req.Method())
			require.Empty(t, req.Scheme())
			require.Empty(t, req.URL())
			require.Empty(t, req.Status())
			require.Empty(t, req.Cookies())
			require.Empty(t, req.SetCookies())
		},
	}, {
		name:    "standard headers set in value",
		headers: envoyHeadersValue,
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, "https", req.Scheme())
			require.Equal(t, "example.com", req.Authority())
			require.Equal(t, "GET", req.Method())
			require.Equal(t, "/", req.URL().Path)
			require.Equal(t, "a", req.URL().Query().Get("q"))
			require.Equal(t, requestID, req.RequestID())
			require.Empty(t, req.Status())
		},
	}, {
		name:    "standard headers set in raw_value",
		headers: envoyHeadersRawValue,
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, "https", req.Scheme())
			require.Equal(t, "example.com", req.Authority())
			require.Equal(t, "GET", req.Method())
			require.Equal(t, "/", req.URL().Path)
			require.Equal(t, "a", req.URL().Query().Get("q"))
			require.Equal(t, requestID, req.RequestID())
			require.Empty(t, req.Status())
		},
	}, {
		name: "path without query",
		headers: []*corev3.HeaderValue{
			{Key: ":scheme", Value: "https"},
			{Key: ":authority", Value: "example.com"},
			{Key: ":path", Value: "/api"},
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, req.URL().Path, "/api")
			require.Equal(t, req.URL().Query().Get("q"), "")
		},
	}, {
		name: "parser cookie header",
		headers: []*corev3.HeaderValue{{
			Key:      "cookie",
			RawValue: []byte("visitor_id=XTIDIOIFXGH5426LOUVSOVB7F0384QAF;locale_code=en-GB"),
		}},
		assert: func(t *testing.T, req *filter.RequestContext) {
			var (
				visitorIDName     = "visitor_id"
				localeCodeName    = "locale_code"
				missingCookieName = "missing_cookie"
			)
			require.Len(t, req.Cookies(), 2)
			visitorID, visitorIDExists := req.GetCookie(visitorIDName)
			require.True(t, visitorIDExists)
			require.Equal(t, visitorID.Name, visitorIDName)
			require.Equal(t, visitorID.Value, "XTIDIOIFXGH5426LOUVSOVB7F0384QAF")

			localeCode, localeCodeExists := req.GetCookie(localeCodeName)
			require.True(t, localeCodeExists)
			require.Equal(t, localeCode.Name, localeCodeName)
			require.Equal(t, localeCode.Value, "en-GB")

			missingCookie, missingCookieExists := req.GetCookie(missingCookieName)
			require.False(t, missingCookieExists)
			require.Equal(t, missingCookie.Name, "")
			require.Equal(t, missingCookie.Value, "")

			require.Equal(t, visitorIDName, req.Cookies()[0].Name)
			require.Equal(t, "XTIDIOIFXGH5426LOUVSOVB7F0384QAF", req.Cookies()[0].Value)

			require.Equal(t, localeCodeName, req.Cookies()[1].Name)
			require.Equal(t, "en-GB", req.Cookies()[1].Value)
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			msg := &extproc.ProcessingRequest_RequestHeaders{
				RequestHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: tt.headers,
					},
				},
			}
			req := &filter.RequestContext{}
			headers := make(http.Header)
			req.Process(msg, headers)
			tt.assert(t, req)
		})
	}
}

func TestProcessResponseHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers []*corev3.HeaderValue
		assert  func(t *testing.T, req *filter.RequestContext)
	}{{
		name:    "empty headers",
		headers: []*corev3.HeaderValue{},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, req.RequestPhase(), filter.RequestPhaseResponseHeaders)
			require.Empty(t, req.RequestID())
			require.Empty(t, req.Authority())
			require.Empty(t, req.Method())
			require.Empty(t, req.Scheme())
			require.Empty(t, req.URL().Host)
			require.Empty(t, req.Status())
			require.Empty(t, req.Cookies())
			require.Empty(t, req.SetCookies())
		},
	}, {
		name: "standard headers set in value",
		headers: []*corev3.HeaderValue{{
			Key:   ":status",
			Value: "200",
		}},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, 200, req.Status())
		},
	}, {
		name: "standard headers set in raw_value",
		headers: []*corev3.HeaderValue{{
			Key:      ":status",
			RawValue: []byte("200"),
		}},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, 200, req.Status())
		},
	}, {
		name: "parser set-cookie header",
		headers: []*corev3.HeaderValue{{
			Key:      "set-cookie",
			RawValue: []byte("locale_code=en-US; path=/; expires=Sun, 11 Feb 2029 23:07:28 GMT; secure"),
		}, {
			Key:   "set-cookie",
			Value: "cur=EUR; path=/; expires=Sun, 11 Feb 2029 23:07:28 GMT; secure",
		}},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Len(t, req.SetCookies(), 2)

			require.Equal(t, "locale_code", req.SetCookies()[0].Name)
			require.Equal(t, "en-US", req.SetCookies()[0].Value)

			require.Equal(t, "cur", req.SetCookies()[1].Name)
			require.Equal(t, "EUR", req.SetCookies()[1].Value)
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			msg := &extproc.ProcessingRequest_ResponseHeaders{
				ResponseHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: tt.headers,
					},
				},
			}
			req := &filter.RequestContext{}
			headers := make(http.Header)
			req.Process(msg, headers)
			tt.assert(t, req)
		})
	}
}

func TestDifferentRequestPhase(t *testing.T) {
	for _, tt := range []struct {
		name string
		msg  any
	}{{
		name: "request headers",
		msg:  &extproc.ProcessingRequest_RequestHeaders{},
	}, {
		name: "request body",
		msg:  &extproc.ProcessingRequest_RequestBody{},
	}, {
		name: "request trailers",
		msg:  &extproc.ProcessingRequest_RequestTrailers{},
	}, {
		name: "response headers",
		msg:  &extproc.ProcessingRequest_ResponseHeaders{},
	}, {
		name: "response body",
		msg:  &extproc.ProcessingRequest_ResponseBody{},
	}, {
		name: "response trailers",
		msg:  &extproc.ProcessingRequest_ResponseTrailers{},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			req := &filter.RequestContext{}
			headers := make(http.Header)
			req.Process(tt.msg, headers)
			req.Authority()
			req.Cookies()
			req.Method()
			req.RequestID()
			req.Scheme()
			req.Status()
			req.Metadata().Get("key")
			req.GetCookie("cookie")
			req.RequestHeader("header")
			req.RequestHeaderValues("header")
			req.ResponseHeader("header")
			req.ResponseHeaderValues("header")
			_ = req.URL().User.String()
		})
	}
}
