package negotiator_test

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rickb777/negotiator"

	"github.com/stretchr/testify/assert"
)

func TestXMLShouldProcessAcceptHeader(t *testing.T) {
	var acceptTests = []struct {
		acceptheader string
		expected     bool
	}{
		{"application/xml", true},
		{"application/xml-dtd", true},
		{"application/CEA", false},
		{"image/svg+xml", true},
	}

	processor := negotiator.XMLProcessor()

	for _, tt := range acceptTests {
		result := processor.CanProcess(tt.acceptheader, "")
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestXMLShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.XMLProcessor()

	processor.Process(recorder, nil, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestXMLShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{
		"Joe Bloggs",
	}

	processor := negotiator.XMLProcessor()

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "application/xml", recorder.HeaderMap.Get("Content-Type"))
}

func TestXMLShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{Name: "Joe Bloggs"}

	processor := negotiator.XMLProcessor().(negotiator.ContentTypeSettable).SetContentType("image/svg+xml")

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "image/svg+xml", recorder.HeaderMap.Get("Content-Type"))
}

func TestXMLShouldSetResponseBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{
		"Joe Bloggs",
	}

	processor := negotiator.XMLProcessor()

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "<ValidXMLUser><Name>Joe Bloggs</Name></ValidXMLUser>", recorder.Body.String())
}

func TestXMlShouldSetResponseBodyWithIndentation(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{Name: "Joe Bloggs"}

	processor := negotiator.IndentedXMLProcessor("  ")

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "<ValidXMLUser>\n  <Name>Joe Bloggs</Name>\n</ValidXMLUser>\n", recorder.Body.String())
}

func TestXMLShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &XMLUser{Name: "Joe Bloggs"}

	processor := negotiator.IndentedXMLProcessor("  ")

	err := processor.Process(recorder, nil, model, "")

	assert.Error(t, err)
}

type ValidXMLUser struct {
	Name string
}

type XMLUser struct {
	Name string
}

func (u *XMLUser) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return errors.New("oops")
}

func xmltestErrorHandler(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
