// package processor defines what a ResponseProcessor is, and provides four standard implementations:
// JSON, XML, CSV and plain text.
package processor

import "net/http"

// ResponseProcessor interface creates the contract for custom content negotiation.
type ResponseProcessor interface {
	// CanProcess is the predicate that determines whether this processor
	// will handle a given request.
	CanProcess(mediaRange string, lang string) bool
	// ContentType returns the content type for this response.
	ContentType() string
	// Process renders the data model to the response writer, without setting any headers.
	// If the processor encounters an error, it should panic.
	Process(w http.ResponseWriter, template string, dataModel interface{})
}

// ContentTypeSettable interface provides for those response processors that allow the
// response Content-Type to be set explicitly.
type ContentTypeSettable interface {
	WithContentType(contentType string) ResponseProcessor
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
