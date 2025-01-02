package test_test

import (
	"testing"

	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTest(t *testing.T) {
	templateData := struct {
		HeaderName  string
		HeaderValue string
	}{
		HeaderName:  "x-custom-header",
		HeaderValue: "value-1",
	}
	testcases := extproctest.LoadTemplate(t, "testdata/httptest.yml", templateData)
	require.NotEmpty(t, testcases)
	testcases.Run(t)
}
