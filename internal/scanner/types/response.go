package types

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
)

var _ Response = (*GoHTTPResponse)(nil)
var _ Response = (*ResponseMeta)(nil)

// Response interface contains general methods for retrieving response info.
type Response interface {
	// GetStatusCode returns response status code.
	GetStatusCode() int

	// GetReason return response status message
	// corresponding to the HTTP status code.
	GetReason() string

	// GetHeaders returns response headers.
	GetHeaders() http.Header

	// GetContent returns response content body.
	GetContent() []byte

	// GetError returns any error that occurred during the request processing.
	GetError() error
}

// GoHTTPResponse is a wrapper that provides implementation of the Response
// interface for the *http.Response.
type GoHTTPResponse struct {
	Resp *http.Response
}

func (r *GoHTTPResponse) GetStatusCode() int {
	return r.Resp.StatusCode
}

func (r *GoHTTPResponse) GetReason() string {
	reasonIndex := strings.Index(r.Resp.Status, " ")
	reason := r.Resp.Status[reasonIndex+1:]
	return reason
}

func (r *GoHTTPResponse) GetHeaders() http.Header {
	return r.Resp.Header
}

func (r *GoHTTPResponse) GetContent() []byte {
	body, err := io.ReadAll(r.Resp.Body)
	if err != nil {
		return nil
	}

	// body reuse
	r.Resp.Body.Close()
	r.Resp.Body = io.NopCloser(bytes.NewReader(body))

	return body
}

func (r *GoHTTPResponse) GetError() error {
	return nil
}

// ResponseMeta provides implementation information about response performed with
// Chrome HTTP client
type ResponseMeta struct {
	StatusCode   int
	StatusReason string
	Headers      http.Header
	Content      []byte
	Error        string
}

func (r *ResponseMeta) GetStatusCode() int {
	return r.StatusCode
}

func (r *ResponseMeta) GetReason() string {
	return r.StatusReason
}

func (r *ResponseMeta) GetHeaders() http.Header {
	return r.Headers
}

func (r *ResponseMeta) GetContent() []byte {
	return r.Content
}

func (r *ResponseMeta) GetError() error {
	if r.Error != "" {
		return errors.New(r.Error)
	}

	return nil
}
