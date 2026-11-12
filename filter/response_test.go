package filter_test

import (
	"testing"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/stretchr/testify/require"
)

// Envoy's ext_proc only appends a header when the deprecated append field is true. When it is unset,
// Envoy overwrites, whatever append_action says, so every header but the last with that name is lost.
func TestImmediateResponseHeaderAppend(t *testing.T) {
	for _, tt := range []struct {
		name       string
		build      func(irw *filter.ImmediateResponseWriter) *filter.ImmediateResponseWriter
		wantAppend []bool
	}{{
		name: "append header sets append",
		build: func(irw *filter.ImmediateResponseWriter) *filter.ImmediateResponseWriter {
			return irw.AppendHeader("set-cookie", "a=1").AppendHeader("set-cookie", "b=2")
		},
		wantAppend: []bool{true, true},
	}, {
		name: "set header does not set append",
		build: func(irw *filter.ImmediateResponseWriter) *filter.ImmediateResponseWriter {
			return irw.SetHeader("location", "/a")
		},
		wantAppend: []bool{false},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.build(filter.NewImmediateResponseBuilder()).ImmediateResponse()
			var gotAppend []bool
			for _, h := range res.ImmediateResponse.GetHeaders().GetSetHeaders() {
				gotAppend = append(gotAppend, h.GetAppend().GetValue()) //nolint:staticcheck // Append is what Envoy reads, see above.
			}
			require.Equal(t, tt.wantAppend, gotAppend)
		})
	}
}
