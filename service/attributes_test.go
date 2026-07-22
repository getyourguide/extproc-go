package service

import (
	"testing"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestMergeAttributesIntoReq(t *testing.T) {
	t.Run("populates empty req.Attributes", func(t *testing.T) {
		req := filter.NewRequestContext()
		mergeAttributesIntoReq(req, map[string]*structpb.Struct{
			"source": {Fields: map[string]*structpb.Value{"address": structpb.NewStringValue("10.0.0.1")}},
		})
		v, ok := req.Attribute("source", "address")
		require.True(t, ok)
		require.Equal(t, "10.0.0.1", v.GetStringValue())
	})

	t.Run("merges fields across calls instead of overwriting the namespace", func(t *testing.T) {
		req := filter.NewRequestContext()
		mergeAttributesIntoReq(req, map[string]*structpb.Struct{
			"request": {Fields: map[string]*structpb.Value{"path": structpb.NewStringValue("/foo")}},
		})
		mergeAttributesIntoReq(req, map[string]*structpb.Struct{
			"request": {Fields: map[string]*structpb.Value{"method": structpb.NewStringValue("GET")}},
		})

		path, ok := req.Attribute("request", "path")
		require.True(t, ok)
		require.Equal(t, "/foo", path.GetStringValue())

		method, ok := req.Attribute("request", "method")
		require.True(t, ok)
		require.Equal(t, "GET", method.GetStringValue())
	})

	t.Run("empty merge is a no-op", func(t *testing.T) {
		req := filter.NewRequestContext()
		mergeAttributesIntoReq(req, nil)
		_, ok := req.Attribute("request", "path")
		require.False(t, ok)
	})
}
