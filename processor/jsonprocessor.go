package processor

import (
	"encoding/json"
	"net/http"
	"strings"
)

const defaultJSONContentType = "application/json"

type jsonProcessor struct {
	dense          bool
	prefix, indent string
	contentType    string
}

// JSON creates a new processor for JSON without indentation.
func JSON() ResponseProcessor {
	return &jsonProcessor{true, "", "", defaultJSONContentType}
}

// IndentedJSON creates a new processor for JSON with a specified indentation.
func IndentedJSON(indent string) ResponseProcessor {
	return &jsonProcessor{false, "", indent, defaultJSONContentType}
}

func (p *jsonProcessor) ContentType() string {
	return p.contentType
}

// WithContentType implements ContentTypeSettable for this type.
func (p *jsonProcessor) WithContentType(contentType string) ResponseProcessor {
	p.contentType = contentType
	return p
}

// Implements AjaxResponseProcessor for this type.
func (*jsonProcessor) IsAjaxResponder() bool {
	return true
}

func (*jsonProcessor) CanProcess(mediaRange string, lang string) bool {
	return strings.EqualFold(mediaRange, "application/json") ||
		strings.HasPrefix(mediaRange, "application/json-") ||
		strings.HasSuffix(mediaRange, "+json")
}

func (p *jsonProcessor) Process(w http.ResponseWriter, _ string, dataModel interface{}) error {
	if p.dense {
		return json.NewEncoder(w).Encode(dataModel)
	}

	js, err := json.MarshalIndent(dataModel, p.prefix, p.indent)

	if err != nil {
		return err
	}

	return WriteWithNewline(w, js)
}
