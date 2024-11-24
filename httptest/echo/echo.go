package echo

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type RequestHeaderResponse struct {
	Headers map[string]string `json:"headers"`
}

// RequestHeaders writes the request headers in the payload
func RequestHeaders(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("content-type", "application/json")
	resp := RequestHeaderResponse{
		Headers: make(map[string]string),
	}
	for headerName := range request.Header {
		resp.Headers[headerName] = request.Header.Get(headerName)
	}

	resp.Headers["Host"] = request.Host
	resp.Headers["Method"] = request.Method

	respond(w, http.StatusOK, resp)
}

type ResponseHeaderResponse map[string]string

// ResponseHeaders writes response headers from query parameters.
func ResponseHeaders(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("content-type", "application/json")
	resp := make(ResponseHeaderResponse)
	for k, v := range request.URL.Query() {
		if len(v) <= 0 {
			continue
		}
		if k == "status" {
			statusCode, err := strconv.Atoi(v[0])
			if err != nil {
				respond(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
				return
			}
			w.WriteHeader(statusCode)
		}
		for _, value := range v {
			w.Header().Add(k, value)
		}
		resp[k] = v[0]
	}
	respond(w, http.StatusOK, resp)
}

func respond(w http.ResponseWriter, statusCode int, v any) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(raw) // nolint:errcheck
}
