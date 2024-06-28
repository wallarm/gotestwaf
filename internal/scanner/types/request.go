package types

import (
	"net/http"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var _ Request = (*GoHTTPRequest)(nil)
var _ Request = (*ChromeDPTasks)(nil)

// Request interface represents either GoHTTPRequest or ChromeDPTasks.
type Request interface {
	// IsRequest is a dummy method to tag a struct
	// as implementing a Request interface.
	IsRequest()
}

// GoHTTPRequest is a type wrapper for the *http.Request.
type GoHTTPRequest struct {
	Req *http.Request

	DebugHeaderValue string
}

func (r *GoHTTPRequest) IsRequest() {}

// ChromeDPTasks is a type wrapper for the chromedp.Tasks.
type ChromeDPTasks struct {
	Tasks chromedp.Tasks

	UserAgentHeader network.Headers

	ResponseMeta     *ResponseMeta
	DebugHeaderValue string
}

func (r *ChromeDPTasks) IsRequest() {}
