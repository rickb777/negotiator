package negotiator

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_should_add_custom_response_processors(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := NewWithJSONAndXML(fakeResponseProcessor)

	g.Expect(len(negotiator.processors)).To(Equal(3))
}

func Test_should_add_custom_response_processors2(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New().Add(NewJSON(), NewXML()).Add(fakeResponseProcessor)

	g.Expect(len(negotiator.processors)).To(Equal(3))
}

func Test_should_add_custom_response_processors_to_beginning(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := NewWithJSONAndXML(fakeResponseProcessor)

	firstProcessor := negotiator.processors[0]
	processorName := reflect.TypeOf(firstProcessor).String()

	g.Expect(processorName).To(Equal("*negotiator.fakeProcessor"))
}

//-------------------------------------------------------------------------------------------------

func Test_should_use_default_processor_if_no_accept_header(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	negotiator.Negotiate(recorder, req, Offer{Data: "foo"})

	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("foo"))
}

func Test_should_give_JSON_response_for_ajax_requests(t *testing.T) {
	g := NewGomegaWithT(t)
	negotiator := NewWithJSONAndXML().WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(xRequestedWith, xmlHttpRequest)
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{"Joe Bloggs"}
	negotiator.Negotiate(recorder, req, Offer{Data: model})

	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("{\"Name\":\"Joe Bloggs\"}\n"))
}

func Test_should_return_406_if_no_matching_accept_header(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "image/png")
	recorder := httptest.NewRecorder()

	negotiator.Negotiate(recorder, req, Offer{Data: "foo"})

	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func xTest_should_return_406_if_no_accept_header_without_custom_response_processor(t *testing.T) {
	g := NewGomegaWithT(t)
	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	NegotiateWithJSONAndXML(recorder, req, Offer{Data: "foo", MediaType: "text/plain"})

	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func Test_should_not_accept_when_explicitly_excluded1(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but application/negotiatortesting"
	req.Header.Add("Accept", "application/negotiatortesting;q=0, */*") // excluded
	req.Header.Add("Accept-Language", "en")                            // accepted
	recorder := httptest.NewRecorder()

	negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "application/negotiatortesting", Language: "en"})

	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func Test_should_not_accept_when_explicitly_excluded2(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/negotiatortesting, */*") // accepted
	// this header means "anything but en"
	req.Header.Add("Accept-Language", "en;q=0, *") // excluded
	recorder := httptest.NewRecorder()

	negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "application/negotiatortesting", Language: "en"})

	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func Test_should_negotiate_and_write_to_response_body(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/negotiatortesting")
	req.Header.Add("Accept-Language", "en") // accepted
	recorder := httptest.NewRecorder()

	negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "application/negotiatortesting"})

	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("foo"))
}

func Test_should_negotiate_a_default_processor(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "*/*")

	recorder := httptest.NewRecorder()
	negotiator.Negotiate(recorder, req, Offer{Data: "foo"})
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("foo"))

	recorder = httptest.NewRecorder()
	negotiator.Negotiate(recorder, req, Offer{Data: "bar", MediaType: "application/negotiatortesting"})
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("bar"))
}

type fakeProcessor struct{}

func (*fakeProcessor) CanProcess(mediaRange string, lang string) bool {
	return mediaRange == "application/negotiatortesting" && (lang == "" || lang == "en")
}

func (*fakeProcessor) Process(w http.ResponseWriter, req *http.Request, model interface{}, _ string) error {
	w.Write([]byte(model.(string)))
	return nil
}

func testLogger(t *testing.T) Printer {
	return func(level byte, message string, data map[string]interface{}) {
		buf := &strings.Builder{}
		fmt.Fprintf(buf, "%c: %s", level, message)
		for k, v := range data {
			fmt.Fprintf(buf, ", %q: %v", k, v)
		}
		log.Printf(buf.String())
	}
}
