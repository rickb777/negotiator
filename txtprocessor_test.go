package negotiator_test

import (
	"net/http/httptest"
	"testing"

	"github.com/rickb777/negotiator"

	"github.com/stretchr/testify/assert"
)

func TestTXTShouldProcessAcceptHeader(t *testing.T) {
	var acceptTests = []struct {
		acceptheader string
		expected     bool
	}{
		{"text/plain", true},
		{"text/*", true},
		{"text/csv", false},
	}

	processor := negotiator.TXTProcessor()

	for _, tt := range acceptTests {
		result := processor.CanProcess(tt.acceptheader, "")
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestTXTShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.TXTProcessor()

	processor.Process(recorder, nil, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestTXTShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.TXTProcessor()

	processor.Process(recorder, nil, "Joe Bloggs", "")

	assert.Equal(t, "text/plain", recorder.HeaderMap.Get("Content-Type"))
}

func TestTXTShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.TXTProcessor().(negotiator.ContentTypeSettable).SetContentType("text/rtf")

	processor.Process(recorder, nil, "Joe Bloggs", "")

	assert.Equal(t, "text/rtf", recorder.HeaderMap.Get("Content-Type"))
}

func TestTXTShouldSetResponseBody(t *testing.T) {
	models := []struct {
		stuff    interface{}
		expected string
	}{
		{"Joe Bloggs", "Joe Bloggs\n"},
		{hidden{tt(2001, 10, 31)}, "(2001-10-31)\n"},
		{tm{"Joe Bloggs"}, "Joe Bloggs\n"},
	}

	processor := negotiator.TXTProcessor()

	for _, m := range models {
		recorder := httptest.NewRecorder()
		err := processor.Process(recorder, nil, m.stuff, "")
		assert.NoError(t, err)
		assert.Equal(t, m.expected, recorder.Body.String())
	}
}

func TestTXTShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := negotiator.TXTProcessor()

	err := processor.Process(recorder, nil, make(chan int, 0), "")

	assert.Error(t, err)
}

type tm struct {
	s string
}

func (tm tm) MarshalText() (text []byte, err error) {
	return []byte(tm.s), nil
}
