package processor_test

import (
	"net/http/httptest"
	"testing"

	"github.com/rickb777/negotiator/processor"
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

	p := processor.TXT()

	for _, tt := range acceptTests {
		result := p.CanProcess(tt.acceptheader, "")
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestTXTShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	p := processor.TXT()

	p.Process(recorder, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestTXTShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	p := processor.TXT()

	p.Process(recorder, "Joe Bloggs", "")

	assert.Equal(t, "text/plain", recorder.HeaderMap.Get("Content-Type"))
}

func TestTXTShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	p := processor.TXT().(processor.ContentTypeSettable).SetContentType("text/rtf")

	p.Process(recorder, "Joe Bloggs", "")

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

	p := processor.TXT()

	for _, m := range models {
		recorder := httptest.NewRecorder()
		err := p.Process(recorder, m.stuff, "")
		assert.NoError(t, err)
		assert.Equal(t, m.expected, recorder.Body.String())
	}
}

func TestTXTShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	p := processor.TXT()

	err := p.Process(recorder, make(chan int, 0), "")

	assert.Error(t, err)
}

type tm struct {
	s string
}

func (tm tm) MarshalText() (text []byte, err error) {
	return []byte(tm.s), nil
}
