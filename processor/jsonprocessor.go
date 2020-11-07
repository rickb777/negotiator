package processor

import (
	"encoding/json"
	"net/http"
	"strings"
)

const defaultJSONContentType = "application/json"

type jsonProcessor struct {
	indent      string
	contentType string
}

// JSON creates a new processor for JSON with a specified indentation.
// It handles all requests except Ajax requests.
func JSON(indent ...string) ResponseProcessor {
	if len(indent) == 0 {
		return &jsonProcessor{contentType: defaultJSONContentType}
	}
	return &jsonProcessor{indent: indent[0], contentType: defaultJSONContentType}
}

func (p *jsonProcessor) ContentType() string {
	return p.contentType
}

// WithContentType implements ContentTypeSettable for this type.
func (p *jsonProcessor) WithContentType(contentType string) ResponseProcessor {
	p.contentType = contentType
	return p
}

func (*jsonProcessor) CanProcess(mediaRange string, lang string) bool {
	return strings.EqualFold(mediaRange, "application/json") ||
		strings.HasPrefix(mediaRange, "application/json-") ||
		strings.HasSuffix(mediaRange, "+json")
}

func (p *jsonProcessor) Process(w http.ResponseWriter, template string, dataModel interface{}) {
	RenderJSON(p.indent)(w, template, dataModel)
}

// RenderJSON returns a rendering function that converts some data into JSON.
func RenderJSON(indent string) func(http.ResponseWriter, string, interface{}) {
	if indent == "" {
		return func(w http.ResponseWriter, _ string, dataModel interface{}) {
			must(json.NewEncoder(w).Encode(dataModel))
		}
	}

	return func(w http.ResponseWriter, _ string, dataModel interface{}) {
		js, err := json.MarshalIndent(dataModel, "", indent)
		must(err)

		must(WriteWithNewline(w, js))
	}
}
