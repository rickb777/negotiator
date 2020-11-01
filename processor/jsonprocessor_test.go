package processor_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rickb777/negotiator/processor"
	"github.com/stretchr/testify/assert"
)

func TestJSONShouldProcessAcceptHeader(t *testing.T) {
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
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestJSONShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	p := processor.JSON()

	p.Process(recorder, nil, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestJSONShouldHandleAjax(t *testing.T) {
	p := processor.JSON()

	assert.True(t, p.(processor.AjaxResponseProcessor).IsAjaxResponder())
}

func TestJSONShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON()

	p.Process(recorder, nil, model, "")

	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestJSONShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON().(processor.ContentTypeSettable).SetContentType("application/calendar+json")

	p.Process(recorder, nil, model, "")

	assert.Equal(t, "application/calendar+json", recorder.HeaderMap.Get("Content-Type"))
}

func TestJSONShouldSetResponseBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.JSON()

	p.Process(recorder, nil, model, "")

	assert.Equal(t, "{\"Name\":\"Joe Bloggs\"}\n", recorder.Body.String())
}

func TestJSONShouldSetResponseBodyWithIndentation(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	p := processor.IndentedJSON("  ")

	p.Process(recorder, nil, model, "")

	assert.Equal(t, "{\n  \"Name\": \"Joe Bloggs\"\n}\n", recorder.Body.String())
}

func TestJSONShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &User{
		"Joe Bloggs",
	}

	p := processor.JSON()

	err := p.Process(recorder, nil, model, "")

	assert.Error(t, err)
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
