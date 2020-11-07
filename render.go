package negotiator

import (
	"net/http"
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

type CodedRender interface {
	Render
	StatusCode() int
}

//-------------------------------------------------------------------------------------------------

type renderer struct {
	data        interface{}
	language    string
	template    string
	contentType string
	process     func(w http.ResponseWriter, template string, dataModel interface{}) error
}

func (r renderer) StatusCode() int {
	return http.StatusOK
}

func (r *renderer) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", r.contentType)
	if r.language != "" {
		w.Header().Set("Content-Language", r.language)
	}
}

func (r *renderer) Render(w http.ResponseWriter) error {
	return r.process(w, r.template, r.data)
}

//-------------------------------------------------------------------------------------------------

type unacceptable struct {
	errorHandler ErrorHandler
}

func (r unacceptable) StatusCode() int {
	return http.StatusNotAcceptable
}

func (r unacceptable) WriteContentType(w http.ResponseWriter) {
	// does nothing
}

func (r unacceptable) Render(w http.ResponseWriter) error {
	r.errorHandler(w, "the accepted formats are not offered by the server", http.StatusNotAcceptable)
	return nil
}

//-------------------------------------------------------------------------------------------------

type emptyCode int

func (r emptyCode) StatusCode() int {
	return int(r)
}

func (r emptyCode) WriteContentType(w http.ResponseWriter) {
	// does nothing
}

func (r emptyCode) Render(w http.ResponseWriter) error {
	return nil
}
