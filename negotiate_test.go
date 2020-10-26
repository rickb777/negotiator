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
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := NewWithJSONAndXML(fakeResponseProcessor)

	g.Expect(len(negotiator.processors)).To(Equal(3))
}

func Test_should_add_custom_response_processors2(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New().Add(NewJSON(), NewXML()).Add(fakeResponseProcessor)

	g.Expect(len(negotiator.processors)).To(Equal(3))
}

func Test_should_add_custom_response_processors_to_beginning(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := NewWithJSONAndXML(fakeResponseProcessor)

	firstProcessor := negotiator.processors[0]
	processorName := reflect.TypeOf(firstProcessor).String()

	g.Expect(negotiator.processors).To(HaveLen(3))
	g.Expect(processorName).To(Equal("*negotiator.fakeProcessor"))
}

//-------------------------------------------------------------------------------------------------

func Test_should_use_default_processor_if_no_accept_header(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/test | foo"))
}

func Test_should_give_JSON_response_for_ajax_requests(t *testing.T) {
	g := NewGomegaWithT(t)
	negotiator := NewWithJSONAndXML().WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(xRequestedWith, xmlHttpRequest)
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{"Joe Bloggs"}
	err := negotiator.Negotiate(recorder, req, Offer{Data: model})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("{\"Name\":\"Joe Bloggs\"}\n"))
}

func Test_should_return_406_if_no_matching_accept_header(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "image/png")
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func Test_should_return_406_when_media_range_is_explicitly_excluded(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but text/test"
	req.Header.Add("Accept", "text/test;q=0, */*") // excluded
	req.Header.Add("Accept-Language", "en")        // accepted
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "text/test", Language: "en"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusNotAcceptable))
}

func Test_should_negotiate_and_write_to_response_body(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/test")
	req.Header.Add("Accept-Language", "en")
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/test | foo"))
}

func Test_should_match_subtype_wildcard(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/*")
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/test | foo"))
}

func Test_should_match_language_wildcard_and_send_content_language_header(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept-Language", "*")
	recorder := httptest.NewRecorder()

	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", Language: "en"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Header().Get("Content-Language")).To(Equal("en"))
	g.Expect(recorder.Body.String()).To(Equal("text/test | foo"))
}

func Test_should_negotiate_a_default_processor(t *testing.T) {
	g := NewGomegaWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	negotiator := New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "*/*")

	recorder := httptest.NewRecorder()
	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/test | foo"))

	recorder = httptest.NewRecorder()
	err = negotiator.Negotiate(recorder, req, Offer{Data: "bar", MediaType: "text/test"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/test | bar"))
}

func Test_should_negotiate_one_of_the_processors(t *testing.T) {
	g := NewGomegaWithT(t)
	var a = &fakeProcessor{match: "text/a"}
	var b = &fakeProcessor{match: "text/b"}
	negotiator := New(a, b).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/a, text/b")

	recorder := httptest.NewRecorder()
	err := negotiator.Negotiate(recorder, req, Offer{Data: "foo", MediaType: "text/a"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/a | foo"))

	recorder = httptest.NewRecorder()
	err = negotiator.Negotiate(recorder, req, Offer{Data: "bar", MediaType: "text/b"})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(recorder.Code).To(Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(Equal("text/b | bar"))
}

//-------------------------------------------------------------------------------------------------

type fakeProcessor struct {
	match string
}

func (p *fakeProcessor) CanProcess(mediaRange string, lang string) bool {
	return mediaRange == p.match && (lang == "" || lang == "en")
}

func (p *fakeProcessor) Process(w http.ResponseWriter, req *http.Request, model interface{}, _ string) error {
	w.Write([]byte(fmt.Sprintf("%s | %v", p.match, model)))
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
