package httptest_test

import (
	"testing"

	test "github.com/getyourguide/extproc-go/httptest"

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
	testcases := test.LoadTemplate(t, "testdata/integration.yml", templateData)
	require.NotEmpty(t, testcases)
	testcases.Run(t)
}
