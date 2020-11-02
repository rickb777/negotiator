package processor

import (
	"encoding"
	"fmt"
	"net/http"
	"strings"
)

const defaultTxtContentType = "text/plain"

type txtProcessor struct {
	contentType string
}

// TXT creates an output processor that serialises strings in text/plain form.
// Model values should be one of the following:
//
// * string
//
// * fmt.Stringer
//
// * encoding.TextMarshaler
func TXT() ResponseProcessor {
	return &txtProcessor{defaultTxtContentType}
}

func (p *txtProcessor) ContentType() string {
	return p.contentType
}

// WithContentType implements ContentTypeSettable for this type.
func (p *txtProcessor) WithContentType(contentType string) ResponseProcessor {
	p.contentType = contentType
	return p
}

func (*txtProcessor) CanProcess(mediaRange string, lang string) bool {
	return strings.EqualFold(mediaRange, "text/plain") || strings.EqualFold(mediaRange, "text/*")
}

func (p *txtProcessor) Process(w http.ResponseWriter, dataModel interface{}, _ string) error {
	s, ok := dataModel.(string)
	if ok {
		return WriteWithNewline(w, []byte(s))
	}

	st, ok := dataModel.(fmt.Stringer)
	if ok {
		return WriteWithNewline(w, []byte(st.String()))
	}

	tm, ok := dataModel.(encoding.TextMarshaler)
	if ok {
		b, err := tm.MarshalText()
		if err != nil {
			return err
		}
		return WriteWithNewline(w, b)
	}

	return fmt.Errorf("Unsupported type for TXT: %T", dataModel)
}
