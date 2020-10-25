package negotiator

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCSVShouldProcessAcceptHeader(t *testing.T) {
	var acceptTests = []struct {
		acceptheader string
		expected     bool
	}{
		{"text/csv", true},
		{"text/*", true},
		{"text/plain", false},
	}

	processor := NewCSV()

	for _, tt := range acceptTests {
		result := processor.CanProcess(tt.acceptheader, "")
		assert.Equal(t, tt.expected, result, "Should process "+tt.acceptheader)
	}
}

func TestCSVShouldReturnNoContentIfNil(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := NewCSV()

	processor.Process(recorder, nil, nil, "")

	assert.Equal(t, 204, recorder.Code)
}

func TestCSVShouldSetDefaultContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := NewCSV()

	processor.Process(recorder, nil, "Joe Bloggs", "")

	assert.Equal(t, "text/csv", recorder.HeaderMap.Get("Content-Type"))
}

func TestCSVShouldSetContentTypeHeader(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := NewCSV().(ContentTypeSettable).SetContentType("text/csv-schema")

	processor.Process(recorder, nil, "Joe Bloggs", "")

	assert.Equal(t, "text/csv-schema", recorder.HeaderMap.Get("Content-Type"))
}

func tt(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestCSVShouldSetResponseBody(t *testing.T) {
	models := []struct {
		stuff    interface{}
		expected string
	}{
		{"Joe Bloggs", "Joe Bloggs\n"},
		{[]string{"Red", "Green", "Blue"}, "Red,Green,Blue\n"},
		{[][]string{{"Red", "Green", "Blue"}, {"Cyan", "Magenta", "Yellow"}}, "Red,Green,Blue\nCyan,Magenta,Yellow\n"},
		{[]int{101, -5, 42}, "101,-5,42\n"},
		{[]int8{101, -5, 42}, "101,-5,42\n"},
		{[]uint{101, 42}, "101,42\n"},
		{[]uint8{101, 42}, "101,42\n"},
		{[][]int{{101, 42}, {39, 7}}, "101,42\n39,7\n"},
		{[][]uint{{101, 42}, {39, 7}}, "101,42\n39,7\n"},
		{Data{"x,y", 9, 4, true}, "\"x,y\",9,4,true\n"},
		{[]Data{{"x", 9, 4, true}, {"y", 7, 1, false}}, "x,9,4,true\ny,7,1,false\n"},
		{[]hidden{{tt(2001, 11, 29)}, {tt(2001, 11, 30)}}, "(2001-11-29),(2001-11-30)\n"},
		{[][]hidden{{{tt(2001, 12, 30)}, {tt(2001, 12, 31)}}}, "(2001-12-30),(2001-12-31)\n"},
		{[]*hidden{{tt(2001, 11, 29)}, {tt(2001, 11, 30)}}, "(2001-11-29),(2001-11-30)\n"},
		{[][]*hidden{{{tt(2001, 12, 30)}, {tt(2001, 12, 31)}}}, "(2001-12-30),(2001-12-31)\n"},
	}

	processor := NewCSV()

	for _, m := range models {
		recorder := httptest.NewRecorder()
		err := processor.Process(recorder, nil, m.stuff, "")
		assert.NoError(t, err)
		assert.Equal(t, m.expected, recorder.Body.String())
	}
}

func TestCSVShouldSetResponseBodyWithTabs(t *testing.T) {
	models := []struct {
		stuff    interface{}
		expected string
	}{
		{"Joe Bloggs", "Joe Bloggs\n"},
		{[]string{"Red", "Green", "Blue"}, "Red\tGreen\tBlue\n"},
		{[][]string{{"Red", "Green", "Blue"}, {"Cyan", "Magenta", "Yellow"}}, "Red\tGreen\tBlue\nCyan\tMagenta\tYellow\n"},
		{[]int{101, -5, 42}, "101\t-5\t42\n"},
		{[]int8{101, -5, 42}, "101\t-5\t42\n"},
		{[]uint{101, 42}, "101\t42\n"},
		{[]uint8{101, 42}, "101\t42\n"},
		{[][]int{{101, 42}, {39, 7}}, "101\t42\n39\t7\n"},
		{[][]uint{{101, 42}, {39, 7}}, "101\t42\n39\t7\n"},
		{Data{"x", 9, 4, true}, "x\t9\t4\ttrue\n"},
		{[]Data{{"x", 9, 4, true}, {"y", 7, 1, false}}, "x\t9\t4\ttrue\ny\t7\t1\tfalse\n"},
	}

	processor := NewCSV('\t')

	for _, m := range models {
		recorder := httptest.NewRecorder()
		err := processor.Process(recorder, nil, m.stuff, "")
		assert.NoError(t, err)
		assert.Equal(t, m.expected, recorder.Body.String())
	}
}

func TestCSVShouldReturnErrorOnError(t *testing.T) {
	recorder := httptest.NewRecorder()

	processor := NewCSV()

	err := processor.Process(recorder, nil, make(chan int, 0), "")

	assert.Error(t, err)
}

type Data struct {
	F1 string
	F2 int
	F3 uint
	F4 bool
}

// has hidden fields
type hidden struct {
	d time.Time
}

func (h hidden) String() string {
	return "(" + h.d.Format("2006-01-02") + ")"
}

//func (u *User) MarshalCSV() ([]byte, error) {
//	return nil, errors.New("oops")
//}
//
//func jsontestErrorHandler(w http.ResponseWriter, err error) {
//	w.WriteHeader(500)
//	w.Write([]byte(err.Error()))
//}
