package types

import (
	"net/http"

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
}

func (r GoHTTPRequest) IsRequest() {}

// ChromeDPTasks is a type wrapper for the chromedp.Tasks.
type ChromeDPTasks struct {
	Tasks chromedp.Tasks
}

func (r ChromeDPTasks) IsRequest() {}
