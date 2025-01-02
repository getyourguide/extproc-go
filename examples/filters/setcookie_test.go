package filters_test

import (
	"testing"

	extproctest "github.com/getyourguide/extproc-go/test"
)

func TestSameSiteLax(t *testing.T) {
	tc := extproctest.Load(t, "testdata/setcookie.yml")
	tc.Run(t)
}
