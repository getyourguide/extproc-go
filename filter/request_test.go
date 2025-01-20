package filter_test

import (
	"net/http"
	"testing"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/stretchr/testify/require"
)

const (
	requestID = "30149a57-c842-9e40-968d-bf9bdbed55b1"
)

var envoyHeadersValue = http.Header{
	":scheme":      []string{"https"},
	":authority":   []string{"example.com"},
	":method":      []string{"GET"},
	":path":        []string{"/?q=a"},
	"X-Request-Id": []string{requestID},
}

func TestMutatedRequestHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers http.Header
		assert  func(t *testing.T, req *filter.RequestContext)
		mutate  func(t *testing.T, crw *filter.CommonResponseWriter)
	}{{
		name:    "set headers",
		headers: envoyHeadersValue,
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.SetHeader("header-a", "value-a")
			crw.SetHeader("header-b", "value-b")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, "value-a", req.RequestHeader("header-a"))
			require.Equal(t, "value-b", req.RequestHeader("header-b"))
		},
	}, {
		name: "append headers",
		headers: func() http.Header {
			headers := envoyHeadersValue.Clone()
			headers.Add("header-c", "value-c")
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.AppendHeader("header-c", "value-c1")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			got := req.RequestHeaderValues("header-c")
			want := []string{"value-c", "value-c1"}
			require.Equal(t, want, got)
		},
	}, {
		name: "remove headers",
		headers: func() http.Header {
			headers := envoyHeadersValue.Clone()
			headers.Add("header-c", "value-c")
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.RemoveHeaders("header-c")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			got := req.RequestHeader("header-c")
			want := ""
			require.Equal(t, want, got)
		},
	}, {
		name: "cross message access",
		headers: func() http.Header {
			headers := envoyHeadersValue.Clone()
			headers.Add("header-c", "value-c")
			return headers
		}(),
		mutate: func(t *testing.T, crw *filter.CommonResponseWriter) {
			crw.SetHeader("header-d", "value-d")
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, "value-d", req.RequestHeader("header-d"))
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			req := &filter.RequestContext{
				RequestHeaders: tt.headers,
			}
			crw := filter.NewCommonResponseWriter(tt.headers)
			tt.mutate(t, crw)
			tt.assert(t, req)
		})
	}
}

func TestMetadata(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		req := filter.NewRequestContext()
		req.Metadata().Set("key", "value")
		require.Equal(t, "value", req.Metadata().Get("key"))
	})

	t.Run("get non-existent key", func(t *testing.T) {
		req := filter.NewRequestContext()
		require.Empty(t, req.Metadata().Get("non-existent"))
	})
}

func TestProcessRequestHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers http.Header
		assert  func(t *testing.T, req *filter.RequestContext)
	}{{
		name: "empty headers",
		assert: func(t *testing.T, req *filter.RequestContext) {
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
		name: "path without query",
		headers: http.Header{
			":scheme":    []string{"https"},
			":authority": []string{"example.com"},
			":path":      []string{"/api"},
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, req.URL().Path, "/api")
			require.Equal(t, req.URL().Query().Get("q"), "")
		},
	}, {
		name: "parser cookie header",
		headers: http.Header{
			"Cookie": []string{"visitor_id=XTIDIOIFXGH5426LOUVSOVB7F0384QAF;locale_code=en-GB"},
		},
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
			req := &filter.RequestContext{
				RequestHeaders: tt.headers,
			}
			tt.assert(t, req)
		})
	}
}

func TestProcessResponseHeaders(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers http.Header
		assert  func(t *testing.T, req *filter.RequestContext)
	}{{
		name: "empty headers",
		assert: func(t *testing.T, req *filter.RequestContext) {
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
		headers: http.Header{
			":status": []string{"200"},
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, 200, req.Status())
		},
	}, {
		name: "standard headers set in raw_value",
		headers: http.Header{
			":status": []string{"200"},
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Equal(t, 200, req.Status())
		},
	}, {
		name: "parser set-cookie header",
		headers: http.Header{
			"Set-Cookie": []string{
				"locale_code=en-US; path=/; expires=Sun, 11 Feb 2029 23:07:28 GMT; secure",
				"cur=EUR; path=/; expires=Sun, 11 Feb 2029 23:07:28 GMT; secure",
			},
		},
		assert: func(t *testing.T, req *filter.RequestContext) {
			require.Len(t, req.SetCookies(), 2)

			require.Equal(t, "locale_code", req.SetCookies()[0].Name)
			require.Equal(t, "en-US", req.SetCookies()[0].Value)

			require.Equal(t, "cur", req.SetCookies()[1].Name)
			require.Equal(t, "EUR", req.SetCookies()[1].Value)
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			req := &filter.RequestContext{
				ResponseHeaders: tt.headers,
			}
			tt.assert(t, req)
		})
	}
}
