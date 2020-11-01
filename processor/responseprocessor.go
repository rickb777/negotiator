// package processor defines what a ResponseProcessor is, and provides four standard implementations:
// JSON, XML, CSV and plain text.
package processor

import "net/http"

// ResponseProcessor interface creates the contract for custom content negotiation.
type ResponseProcessor interface {
	CanProcess(mediaRange string, lang string) bool
	Process(w http.ResponseWriter, req *http.Request, dataModel interface{}, template string) error
}

// ContentTypeSettable interface provides for those response processors that allow the
// response Content-Type to be set explicitly.
type ContentTypeSettable interface {
	SetContentType(contentType string) ResponseProcessor
}

// AjaxResponseProcessor interface allows content negotiation to be biased when
// Ajax requests are handled. If a ResponseProcessor also implements this interface
// and its method returns true, then all Ajax requests will be fulfilled by that
// request processor, instead of via the normal content negotiation.
type AjaxResponseProcessor interface {
	IsAjaxResponder() bool
}
