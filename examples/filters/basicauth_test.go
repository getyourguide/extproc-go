package filters_test

import (
	"testing"

	"github.com/getyourguide/extproc-go/httptest"
)

func TestBasicAuth(t *testing.T) {
	tc := httptest.Load(t, "testdata/basicauth.yml")
	tc.Run(t)
}
