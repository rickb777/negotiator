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

// SetContentType implements ContentTypeSettable for this type.
func (p *jsonProcessor) SetContentType(contentType string) ResponseProcessor {
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

func (p *jsonProcessor) Process(w http.ResponseWriter, req *http.Request, dataModel interface{}, _ string) error {
	if dataModel == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	w.Header().Set("Content-Type", p.contentType)
	if p.dense {
		return json.NewEncoder(w).Encode(dataModel)
	}

	js, err := json.MarshalIndent(dataModel, p.prefix, p.indent)

	if err != nil {
		return err
	}

	return writeWithNewline(w, js)
}
