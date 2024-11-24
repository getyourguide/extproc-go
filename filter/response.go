package filter

import (
	"net/http"
	"slices"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
)

// routerHeaders requires ClearRouteCache to be set to true
var routerHeaders = map[string]struct{}{
	"host":       {},
	":authority": {},
	":path":      {},
	":method":    {},
}

func isRouterHeader(key string) bool {
	_, ok := routerHeaders[key]
	return ok
}

// CommonResponseWriter is a wraper on top of extproc.CommonResponse
// It provides a fluent API to mutate the request and response headers and body
type CommonResponseWriter struct {
	commonResponse *extproc.CommonResponse
	headers        http.Header
}

func NewCommonResponseWriter(headers http.Header) *CommonResponseWriter {
	if headers == nil {
		headers = make(http.Header)
	}
	crw := &CommonResponseWriter{
		commonResponse: &extproc.CommonResponse{
			HeaderMutation: &extproc.HeaderMutation{},
			Trailers:       &corev3.HeaderMap{},
			BodyMutation:   &extproc.BodyMutation{},
		},
		headers: headers,
	}
	return crw
}

// headerAction sets a header with the given key and value and the given append action
func (crw *CommonResponseWriter) headerAction(key string, value string, appendAction corev3.HeaderValueOption_HeaderAppendAction) *CommonResponseWriter {
	var shouldAppend *wrappers.BoolValue
	switch appendAction {
	case corev3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD:
		shouldAppend = &wrappers.BoolValue{Value: true} // FIXME: This is not the documented behavior but it seems to be the only way to append a header.
		crw.headers.Add(key, value)
	case corev3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD, corev3.HeaderValueOption_OVERWRITE_IF_EXISTS:
		crw.headers.Set(key, value)
	}
	crw.commonResponse.HeaderMutation.SetHeaders = append(crw.commonResponse.HeaderMutation.SetHeaders, &corev3.HeaderValueOption{
		Header: &corev3.HeaderValue{
			Key: key,
			// FIXME: This should be configurable.
			// https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ext_proc/v3/external_processor.proto#envoy-v3-api-msg-service-ext-proc-v3-httpheaders
			// The headers encoding is based on the runtime guard envoy_reloadable_features_send_header_raw_value setting.
			// When it is true, the header value is encoded in the raw_value field. When it is false, the header value is encoded in the value field.
			RawValue: []byte(value), // FIXME: This depends on Envoy
		},
		AppendAction: appendAction,
		Append:       shouldAppend,
	})
	if isRouterHeader(key) {
		crw.ClearRouteCache(true)
	}
	return crw
}

// SetHeader sets a header with the given key and value using the OVERWRITE_IF_EXISTS_OR_ADD action
// This action will overwrite the specified value by discarding any existing values if the header already exists. If the header doesn't exist then this will add the header with specified key and value.
func (crw *CommonResponseWriter) SetHeader(key string, value string) *CommonResponseWriter {
	return crw.headerAction(key, value, corev3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD)
}

// AppendHeader appends a header with the given key and value using the APPEND_IF_EXISTS_OR_ADD action
// This action will append the specified value to the existing values if the header already exists. If the header doesn't exist then this will add the header with specified key and value.
func (crw *CommonResponseWriter) AppendHeader(key string, value string) *CommonResponseWriter {
	return crw.headerAction(key, value, corev3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD)
}

// RemoveHeaders removes these HTTP headers. Attempts to remove system headers -- any header starting with “:“, plus “host“ -- will be ignored.
func (crw *CommonResponseWriter) RemoveHeaders(headers ...string) *CommonResponseWriter {
	for _, h := range headers {
		if slices.Contains(crw.commonResponse.HeaderMutation.RemoveHeaders, h) {
			continue
		}
		crw.commonResponse.HeaderMutation.RemoveHeaders = append(crw.commonResponse.HeaderMutation.RemoveHeaders, h)
		crw.headers.Del(h)
	}
	return crw
}

// SetStatus sets the status of the GRPC response.
// If set, provide additional direction on how the Envoy proxy should handle the rest of the HTTP filter chain.
func (crw *CommonResponseWriter) SetStatus(status extproc.CommonResponse_ResponseStatus) *CommonResponseWriter {
	crw.commonResponse.Status = status
	return crw
}

// ClearRouteCache clears the route cache for the current client request. This is necessary if the remote server modified headers that are used to calculate the route. This field is ignored in the response direction.
func (crw *CommonResponseWriter) ClearRouteCache(clear bool) *CommonResponseWriter {
	crw.commonResponse.ClearRouteCache = clear
	return crw
}

// Replace the body of the last message sent to the remote server on this stream.
// If responding to an HttpBody request, simply replace or clear the body chunk that was sent with that request.
// Body mutations may take effect in response either to “header“ or “body“ messages. When it is in response to “header“ messages, it only take effect if the :ref:`status <envoy_v3_api_field_service.ext_proc.v3.CommonResponse.status>` is set to CONTINUE_AND_REPLACE.
func (crw *CommonResponseWriter) BodyMutation(m *extproc.BodyMutation) *CommonResponseWriter {
	crw.commonResponse.BodyMutation = m
	crw.SetStatus(extproc.CommonResponse_CONTINUE_AND_REPLACE)
	return crw
}

// CommonResponse returns the underlying extproc.CommonResponse
func (crw *CommonResponseWriter) CommonResponse() *extproc.CommonResponse {
	return crw.commonResponse
}

// ImmediateResponseWriter is a wraper on top of extproc.ProcessingResponse_ImmediateResponse
type ImmediateResponseWriter struct {
	immediateResponse *extproc.ProcessingResponse_ImmediateResponse
}

func NewImmediateResponseBuilder() *ImmediateResponseWriter {
	return &ImmediateResponseWriter{
		immediateResponse: &extproc.ProcessingResponse_ImmediateResponse{
			ImmediateResponse: &extproc.ImmediateResponse{
				Headers: &extproc.HeaderMutation{
					SetHeaders:    []*corev3.HeaderValueOption{},
					RemoveHeaders: []string{},
				},
			},
		},
	}
}

// headerAction sets a header with the given key and value and the given append action
func (irw *ImmediateResponseWriter) headerAction(key string, value string, appendAction corev3.HeaderValueOption_HeaderAppendAction) *ImmediateResponseWriter {
	irw.immediateResponse.ImmediateResponse.Headers.SetHeaders = append(irw.immediateResponse.ImmediateResponse.Headers.SetHeaders, &corev3.HeaderValueOption{
		Header: &corev3.HeaderValue{
			Key: key,
			// FIXME: This should be configurable.
			// https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ext_proc/v3/external_processor.proto#envoy-v3-api-msg-service-ext-proc-v3-httpheaders
			// The headers encoding is based on the runtime guard envoy_reloadable_features_send_header_raw_value setting.
			// When it is true, the header value is encoded in the raw_value field. When it is false, the header value is encoded in the value field.
			RawValue: []byte(value), // FIXME: This depends on Envoy
		},
		AppendAction: appendAction,
	})
	return irw
}

// SetHeader sets a header with the given key and value using the OVERWRITE_IF_EXISTS_OR_ADD action
// This action will overwrite the specified value by discarding any existing values if the header already exists. If the header doesn't exist then this will add the header with specified key and value.
func (irw *ImmediateResponseWriter) SetHeader(key string, value string) *ImmediateResponseWriter {
	return irw.headerAction(key, value, corev3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD)
}

// AppendHeader appends a header with the given key and value using the APPEND_IF_EXISTS_OR_ADD action
// This action will append the specified value to the existing values if the header already exists. If the header doesn't exist then this will add the header with specified key and value.
func (irw *ImmediateResponseWriter) AppendHeader(key string, value string) *ImmediateResponseWriter {
	return irw.headerAction(key, value, corev3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD)
}

// RemoveHeaders removes these HTTP headers. Attempts to remove system headers -- any header starting with “:“, plus “host“ -- will be ignored.
func (irw *ImmediateResponseWriter) RemoveHeaders(headers ...string) *ImmediateResponseWriter {
	for _, h := range headers {
		if slices.Contains(irw.immediateResponse.ImmediateResponse.Headers.RemoveHeaders, h) {
			continue
		}
		irw.immediateResponse.ImmediateResponse.Headers.RemoveHeaders = append(irw.immediateResponse.ImmediateResponse.Headers.RemoveHeaders, h)
	}
	return irw
}

// HTTPStatus sets the HTTP status of the immediate response
func (irw *ImmediateResponseWriter) HTTPStatus(status int) *ImmediateResponseWriter {
	irw.immediateResponse.ImmediateResponse.Status = &typev3.HttpStatus{
		Code: typev3.StatusCode(status),
	}
	return irw
}

// ImmediateResponse returns the underlying extproc.ProcessingResponse_ImmediateResponse
func (irw *ImmediateResponseWriter) ImmediateResponse() *extproc.ProcessingResponse_ImmediateResponse {
	return irw.immediateResponse
}

// Body sets the body of the immediate response
func (irw *ImmediateResponseWriter) Body(body []byte) *ImmediateResponseWriter {
	irw.immediateResponse.ImmediateResponse.Body = body
	return irw
}
