package processor

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

const defaultXMLContentType = "application/xml; charset=utf-8"

type xmlProcessor struct {
	indent      string
	contentType string
}

// XML creates a new processor for XML without indentation.
func XML() ResponseProcessor {
	return &xmlProcessor{contentType: defaultXMLContentType}
}

// IndentedXML creates a new processor for XML with a specified indentation.
func IndentedXML(index string) ResponseProcessor {
	return &xmlProcessor{indent: index, contentType: defaultXMLContentType}
}

func (p *xmlProcessor) ContentType() string {
	return p.contentType
}

// WithContentType implements ContentTypeSettable for this type.
func (p *xmlProcessor) WithContentType(contentType string) ResponseProcessor {
	p.contentType = contentType
	return p
}

func (*xmlProcessor) CanProcess(mediaRange string, lang string) bool {
	// see https://tools.ietf.org/html/rfc7303 XML Media Types
	return mediaRange == "application/xml" || mediaRange == "text/xml" ||
		strings.HasSuffix(mediaRange, "+xml") ||
		strings.HasPrefix(mediaRange, "application/xml-") ||
		strings.HasPrefix(mediaRange, "text/xml-")
}

func (p *xmlProcessor) Process(w http.ResponseWriter, _ string, dataModel interface{}) {
	must(p.doProcess(w, "", dataModel))
}

func (p *xmlProcessor) doProcess(w http.ResponseWriter, _ string, dataModel interface{}) error {
	if p.indent == "" {
		return xml.NewEncoder(w).Encode(dataModel)
	}

	x, err := xml.MarshalIndent(dataModel, "", p.indent)
	if err != nil {
		return err
	}

	return WriteWithNewline(w, x)
}

// WriteWithNewline is a helper function that writes some bytes to a Writer. If the
// byte slice is empty or if the last byte is *not* newline, an extra newline is
// also written, as required for HTTP responses.
func WriteWithNewline(w io.Writer, x []byte) error {
	_, err := w.Write(x)
	if err != nil {
		return err
	}

	if len(x) == 0 || x[len(x)-1] != '\n' {
		_, err = w.Write([]byte{'\n'})
	}
	return err
}
