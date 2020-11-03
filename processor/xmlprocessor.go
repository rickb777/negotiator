package processor

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

const defaultXMLContentType = "application/xml"

type xmlProcessor struct {
	dense          bool
	prefix, indent string
	contentType    string
}

// XML creates a new processor for XML without indentation.
func XML() ResponseProcessor {
	return &xmlProcessor{true, "", "", defaultXMLContentType}
}

// IndentedXML creates a new processor for XML with a specified indentation.
func IndentedXML(index string) ResponseProcessor {
	return &xmlProcessor{false, "", index, defaultXMLContentType}
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
	return strings.Contains(mediaRange, "/xml") || strings.HasSuffix(mediaRange, "+xml")
}

func (p *xmlProcessor) Process(w http.ResponseWriter, _ string, dataModel interface{}) error {
	if p.dense {
		return xml.NewEncoder(w).Encode(dataModel)
	}

	x, err := xml.MarshalIndent(dataModel, p.prefix, p.indent)
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
