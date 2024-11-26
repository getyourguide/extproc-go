package httptest

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

type TestCases []Case

type Case struct {
	Name   string `json:"name"`
	Input  Input  `json:"input"`
	Expect Expect `json:"expect"`
	retry  Retry
}

type Retry struct {
	MaxAttempts int           // Maximum number of retries
	WaitMin     time.Duration // Minimum time to wait
	WaitMax     time.Duration // Maximum time to wait

	// PostHook specifies a policy for handling retries. It is called
	// following each request with the response and error values returned by
	// the http call. If PostHook returns false, the Client stops retrying
	PostHook func(Actual) bool
}

type Options interface {
	apply(*Case)
}

type optionFunc func(*Case)

func (f optionFunc) apply(v *Case) {
	f(v)
}

func WithRetry(r Retry) Options {
	return optionFunc(func(c *Case) {
		c.retry = r
	})
}

func (c Case) Run(t *testing.T, opts ...Options) {
	for _, opt := range opts {
		opt.apply(&c)
	}
	t.Run(c.Name, func(t *testing.T) {
		var err error
		for attempt := 0; attempt <= c.retry.MaxAttempts; attempt++ {
			got := httpCall(t, c)
			err = c.Expect.Assert(t, got)
			if err == nil {
				break
			}
			if c.retry.PostHook != nil {
				if !c.retry.PostHook(got) {
					break
				}
			}
			mult := math.Pow(2, float64(attempt)) * float64(c.retry.WaitMin)
			sleep := time.Duration(mult)
			if float64(sleep) != mult || sleep > c.retry.WaitMax {
				sleep = c.retry.WaitMax
			}
			t.Logf("test %q failed, attempt %d/%d. Retrying in %v", c.Name, attempt, c.retry.MaxAttempts, sleep)
			time.Sleep(sleep)
		}
		require.NoError(t, err)
	})
}

type Input struct {
	Headers Headers `json:"headers"`
}

type Headers []HeaderValue

func (headers Headers) Get(key string) string {
	for _, h := range headers {
		if h.Key == key {
			return h.Value
		}
	}
	return ""
}

type Actual struct {
	ResponseHeaders http.Header
	RequestHeaders  http.Header
	Body            string
}

type Expect struct {
	RequestHeaders  []HeaderMatch `json:"requestHeaders"`
	ResponseHeaders []HeaderMatch `json:"responseHeaders"`
	ResponseBody    *StringMatch  `json:"responseBody"`
}

func (e Expect) Assert(t *testing.T, actual Actual) error {
	for _, h := range e.RequestHeaders {
		if !h.Assert(t, actual.RequestHeaders) {
			return fmt.Errorf("header match fail: request header %q should match %q header values with %q=%q and its values are %v", *h.Name, cmp.Or(h.MatchAction, MatchActionFirst), h.MatchType(), h.MatchValue(), actual.RequestHeaders.Values(*h.Name))
		}
	}

	for _, h := range e.ResponseHeaders {
		if !h.Assert(t, actual.ResponseHeaders) {
			return fmt.Errorf("header match fail: response header %q should match %q header values with %q=%q and its values are %q", *h.Name, cmp.Or(h.MatchAction, MatchActionFirst), h.MatchType(), h.MatchValue(), actual.ResponseHeaders.Values(*h.Name))
		}
	}
	if e.ResponseBody != nil && !e.ResponseBody.Assert(t, actual.Body) {
		return fmt.Errorf("response body should match %q=%q and its content is \n%q", e.ResponseBody.MatchType(), e.ResponseBody.MatchValue(), actual.Body)
	}
	return nil
}

type HeaderValue struct {
	Key   string `json:"name"`
	Value string `json:"value"`
}

type MatchAction string

const (
	MatchActionFirst MatchAction = "FIRST"
	MatchActionAny   MatchAction = "ANY"
	MatchActionAll   MatchAction = "ALL"
)

// TODO(cainelli): Use the generic StringMatch
type HeaderMatch struct {
	Name        *string     `json:"name"`
	Exact       *string     `json:"exact"`
	Absent      *bool       `json:"absent"`
	Regex       *string     `json:"regex"`
	MatchAction MatchAction `json:"matchAction"`
}

func (hm HeaderMatch) Assert(t *testing.T, headers http.Header) bool {
	switch hm.MatchAction {
	case "", MatchActionFirst:
		headerValue := headers.Get(*hm.Name)
		return hm.match(headerValue)
	case MatchActionAny:
		for _, value := range headers.Values(*hm.Name) {
			if hm.match(value) {
				return true
			}
		}
		return false
	case MatchActionAll:
		headerValues := headers.Values(*hm.Name)
		if len(headerValues) == 0 {
			return false
		}
		for _, value := range headers.Values(*hm.Name) {
			if !hm.match(value) {
				return false
			}
		}
		return true
	}
	return false
}

func (hm *HeaderMatch) match(value string) bool {
	switch {
	case hm.Absent != nil:
		if *hm.Absent {
			return value == ""
		}
		return value != ""
	case hm.Exact != nil:
		return value == *hm.Exact
	case hm.Regex != nil:
		r := regexp.MustCompile(*hm.Regex)
		return r.MatchString(value)
	}
	return false
}

func (hm *HeaderMatch) MatchType() string {
	switch {
	case hm.Exact != nil:
		return "exact"
	case hm.Absent != nil:
		return "absent"
	case hm.Regex != nil:
		return "regex"
	}
	return ""
}

func (hm *HeaderMatch) MatchValue() string {
	switch {
	case hm.Exact != nil:
		return *hm.Exact
	case hm.Absent != nil:
		return fmt.Sprintf("%t", *hm.Absent)
	case hm.Regex != nil:
		return *hm.Regex
	}

	return ""
}

func (cases TestCases) Run(t *testing.T) {
	for _, tt := range cases {
		tt.Run(t)
	}
}

func httpCall(t *testing.T, tt Case) Actual {
	httpClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer httpClient.CloseIdleConnections()

	baseURL := "http://127.0.0.1:10000"
	if endpoint := os.Getenv("EXTPROC_TEST_ENDPOINT"); endpoint != "" {
		baseURL = endpoint
	}
	url := fmt.Sprintf("%s%s", baseURL, tt.Input.Headers.Get("path"))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	for _, header := range tt.Input.Headers {
		if strings.ToLower(header.Key) == "host" {
			req.Host = header.Value
		}
		if strings.ToLower(header.Key) == "method" {
			req.Method = header.Value
		}
		req.Header.Add(header.Key, header.Value)
	}
	res, err := httpClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.True(t, (res.StatusCode > 200 || res.StatusCode < 499), "invalid status code in res from server", "status", res.StatusCode)

	var response struct {
		Headers map[string]string `json:"headers"`
	}
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	requestHeaders := http.Header{}
	if res.StatusCode == 200 && tt.Expect.RequestHeaders != nil {
		rdr := io.NopCloser(bytes.NewBuffer(body))
		err = json.NewDecoder(rdr).Decode(&response)
		require.NoError(t, err, "error decoding response")

		for k, v := range response.Headers {
			requestHeaders.Add(k, v)
		}
	}
	res.Header.Add("status", fmt.Sprintf("%d", res.StatusCode))
	actual := Actual{
		ResponseHeaders: res.Header,
		RequestHeaders:  requestHeaders,
		Body:            string(body),
	}

	return actual
}

func Load(t *testing.T, path string) TestCases {
	if testing.Short() {
		t.Skip()
	}
	return testData(t, nil, path)
}

func LoadTemplate(t *testing.T, path string, templateData any) TestCases {
	if testing.Short() {
		t.Skip()
	}
	return testData(t, templateData, path)
}

func testData(t *testing.T, templateData any, files ...string) TestCases {
	var configs TestCases
	for _, fileName := range files {
		if !strings.Contains(fileName, "testdata/") {
			fileName = fmt.Sprintf("testdata/%s", fileName)
		}

		tmpl, err := template.ParseFiles(fileName)
		require.NoError(t, err)
		b := bytes.NewBuffer([]byte{})
		err = tmpl.Execute(b, templateData)
		require.NoError(t, err)

		for _, doc := range bytes.Split(b.Bytes(), []byte("---")) {
			var testcase Case
			err = yaml.Unmarshal(doc, &testcase)
			require.NoError(t, err)
			configs = append(configs, testcase)
		}
	}

	return configs
}
