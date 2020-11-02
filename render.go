package negotiator

import (
	"net/http"

	"github.com/rickb777/negotiator/processor"
)

// Render defines the interface for content renderers.
// Note that it happens to match render.Render in github.com/gin-gonic/gin/render.
// This means that this negotiator package can be used with Gin directly.
type Render interface {
	// Render writes data with custom ContentType.
	Render(http.ResponseWriter) error
	// WriteContentType writes custom ContentType.
	WriteContentType(w http.ResponseWriter)
}

//-------------------------------------------------------------------------------------------------

type renderer struct {
	data     interface{}
	language string
	template string
	p        processor.ResponseProcessor
}

// WriteContentType writes custom ContentType.
func (r *renderer) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", r.p.ContentType())
	if r.language != "" {
		w.Header().Set("Content-Language", r.language)
	}
}

// Render writes data with custom ContentType.
func (r *renderer) Render(w http.ResponseWriter) error {
	return r.p.Process(w, r.data, r.template)
}

//-------------------------------------------------------------------------------------------------

type unacceptable struct {
	errorHandler ErrorHandler
}

// WriteContentType writes custom ContentType.
func (r unacceptable) WriteContentType(w http.ResponseWriter) {
	// does nothing
}

// Render writes data with custom ContentType.
func (r unacceptable) Render(w http.ResponseWriter) error {
	r.errorHandler(w, "the accepted formats are not offered by the server", http.StatusNotAcceptable)
	return nil
}
