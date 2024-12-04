package filters_test

import (
	"testing"

	"github.com/getyourguide/extproc-go/httptest"
)

func TestSameSiteLax(t *testing.T) {
	tc := httptest.Load(t, "testdata/setcookie.yml")
	tc.Run(t)
}
