package processor_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/rickb777/negotiator/processor"
)

func TestJSONShouldProcessAcceptHeader(t *testing.T) {
	g := NewGomegaWithT(t)
	var acceptTests = []struct {
		acceptheader string
		expected     bool
	}{
		{"application/json", true},
		{"application/json-", true},
		{"application/CEA", false},
		{"+json", true},
	}

	p := processor.JSON()

	for _, tt := range acceptTests {
		result := p.CanProcess(tt.acceptheader, "")

		g.Expect(result).To(Equal(tt.expected), "Should process "+tt.acceptheader)
	}
}

func TestJSONShouldReturnNoContentIfNil(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	p := processor.JSON()

	p.Process(recorder, nil, "")

	g.Expect(recorder.Code).To(Equal(204))
}

func TestJSONShouldHandleAjax(t *testing.T) {
	g := NewGomegaWithT(t)

	p := processor.JSON()

	g.Expect(p.(processor.AjaxResponseProcessor).IsAjaxResponder()).To(BeTrue())
}

func TestJSONShouldSetDefaultContentTypeHeader(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON()

	p.Process(recorder, model, "")

	g.Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))
}

func TestJSONShouldSetContentTypeHeader(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON().(processor.ContentTypeSettable).SetContentType("application/calendar+json")

	p.Process(recorder, model, "")

	g.Expect(recorder.Header().Get("Content-Type")).To(Equal("application/calendar+json"))
}

func TestJSONShouldSetResponseBody(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON()

	p.Process(recorder, model, "")

	g.Expect(recorder.Body.String()).To(Equal("{\"Name\":\"Joe Bloggs\"}\n"))
}

func TestJSONShouldSetResponseBodyWithIndentation(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.IndentedJSON("  ")

	p.Process(recorder, model, "")

	g.Expect(recorder.Body.String()).To(Equal("{\n  \"Name\": \"Joe Bloggs\"\n}\n"))
}

func TestJSONShouldReturnErrorOnError(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := httptest.NewRecorder()

	model := &User{
		"Joe Bloggs",
	}

	p := processor.JSON()

	err := p.Process(recorder, model, "")

	g.Expect(err).To(HaveOccurred())
}

type User struct {
	Name string
}

func (u *User) MarshalJSON() ([]byte, error) {
	return nil, errors.New("oops")
}

func jsontestErrorHandler(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
