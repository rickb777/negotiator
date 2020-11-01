package negotiator_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rickb777/negotiator"

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

	processor := negotiator.JSONProcessor()

	for _, tt := range acceptTests {
		result := processor.CanProcess(tt.acceptheader, "")
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestJSONShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.JSONProcessor()

	processor.Process(recorder, nil, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestJSONShouldHandleAjax(t *testing.T) {
	processor := negotiator.JSONProcessor()

	assert.True(t, processor.(negotiator.AjaxResponseProcessor).IsAjaxResponder())
}

func TestJSONShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	processor := negotiator.JSONProcessor()

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestJSONShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	processor := negotiator.JSONProcessor().(negotiator.ContentTypeSettable).SetContentType("application/calendar+json")

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "application/calendar+json", recorder.HeaderMap.Get("Content-Type"))
}

func TestJSONShouldSetResponseBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	processor := negotiator.JSONProcessor()

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "{\"Name\":\"Joe Bloggs\"}\n", recorder.Body.String())
}

func TestJSONShouldSetResponseBodyWithIndentation(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := struct {
		Name string
	}{
		"Joe Bloggs",
	}

	processor := negotiator.IndentedJSONProcessor("  ")

	processor.Process(recorder, nil, model, "")

	assert.Equal(t, "{\n  \"Name\": \"Joe Bloggs\"\n}\n", recorder.Body.String())
}

func TestJSONShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	model := &User{
		"Joe Bloggs",
	}

	processor := negotiator.JSONProcessor()

	err := processor.Process(recorder, nil, model, "")

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
