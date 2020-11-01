package negotiator_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/rickb777/negotiator"
	"github.com/rickb777/negotiator/processor"
)

type User struct {
	Name string
}

// Negotiate applies the negotiation algorithm, choosing the response
// based on the Accept header in the request, if present.
// It returns either a successful response, a 406-Not Acceptable,
// or possibly a 500-Internal server error.
//
// In this example, there is only one offer and it will be used by whichever
// response processor matches the request.
func ExampleNegotiator_Negotiate_singleOffer() {
	// getUser is a 'standard' handler function
	getUser := func(w http.ResponseWriter, req *http.Request) {
		// some data; this will be wrapped in an Offer{}
		user := &User{Name: "Joe Bloggs"}

		// the negotiator determines the response format based on the request headers
		negotiator.Default().Negotiate(w, req, negotiator.Offer{Data: user})
	}

	// normal handling
	http.Handle("/user", http.HandlerFunc(getUser))
}

func Test_should_add_custom_response_processors(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.Default().Insert(fakeResponseProcessor)

	g.Expect(n.N()).To(gomega.Equal(5))
}

func Test_should_add_custom_response_processors2(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New().Append(processor.JSON(), processor.XML()).Append(fakeResponseProcessor)

	g.Expect(n.N()).To(gomega.Equal(3))
}

func Test_should_add_custom_response_processors_to_beginning(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.Default().Insert(fakeResponseProcessor)

	firstProcessor := n.Processor(0)
	processorName := reflect.TypeOf(firstProcessor).String()

	g.Expect(n.N()).To(gomega.Equal(5))
	g.Expect(processorName).To(gomega.Equal("*negotiator_test.fakeProcessor"))
}

//-------------------------------------------------------------------------------------------------

func Test_should_use_default_processor_if_no_accept_header(t *testing.T) {
	g := gomega.NewWithT(t)
	var a = &fakeProcessor{match: "text/test"}
	var b = &fakeProcessor{match: "text/plain"}
	n := negotiator.New(a, b).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_give_JSON_response_for_ajax_requests(t *testing.T) {
	g := gomega.NewWithT(t)
	n := negotiator.Default().WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(negotiator.XRequestedWith, negotiator.XMLHttpRequest)
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{Name: "Joe Bloggs"}
	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: model})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("{\"Name\":\"Joe Bloggs\"}\n"))
}

func Test_should_return_406_if_no_matching_accept_header(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	cases := []string{"*/*", "text/test"}

	for _, c := range cases {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Add("Accept", "image/png")
		recorder := httptest.NewRecorder()

		err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: c})

		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(recorder.Code).To(gomega.Equal(http.StatusNotAcceptable))
	}
}

// RFC7231 suggests that 406 is sent when no media range matches are possible.
func Test_should_return_406_when_media_range_is_explicitly_excluded(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but text/test"
	req.Header.Add("Accept", "text/test;q=0, */*") // excluded
	req.Header.Add("Accept-Language", "en")        // accepted
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test", Language: "en"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusNotAcceptable))
}

// RFC7231 recommends that, when no language matches are possible  a response should be sent anyway.
func Test_should_return_200_even_when_language_is_explicitly_excluded(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but text/test"
	req.Header.Add("Accept", "text/test, */*")
	req.Header.Add("Accept-Language", "en;q=0 *") // anything but "en"
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test", Language: "en"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
}

func Test_should_negotiate_and_write_to_response_body(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/test")
	req.Header.Add("Accept-Language", "en")
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_match_subtype_wildcard(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/*")
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_match_language_wildcard_and_send_content_language_header(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept-Language", "*")
	recorder := httptest.NewRecorder()

	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", Language: "en"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Header().Get("Content-Language")).To(gomega.Equal("en"))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_negotiate_a_default_processor(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "*/*")

	recorder := httptest.NewRecorder()
	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))

	recorder = httptest.NewRecorder()
	err = n.TryNegotiate(recorder, req, negotiator.Offer{Data: "bar", MediaType: "text/test"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | bar"))
}

func Test_should_negotiate_one_of_the_processors(t *testing.T) {
	g := gomega.NewWithT(t)
	var a = &fakeProcessor{match: "text/a"}
	var b = &fakeProcessor{match: "text/b"}
	n := negotiator.New(a, b).WithLogger(testLogger(t))

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/a, text/b")

	recorder := httptest.NewRecorder()
	err := n.TryNegotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/a"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/a | foo"))

	recorder = httptest.NewRecorder()
	err = n.TryNegotiate(recorder, req, negotiator.Offer{Data: "bar", MediaType: "text/b"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/b | bar"))
}

//-------------------------------------------------------------------------------------------------

type fakeProcessor struct {
	match string
}

func (p *fakeProcessor) CanProcess(mediaRange string, lang string) bool {
	return mediaRange == p.match && (lang == "*" || lang == "en")
}

func (p *fakeProcessor) Process(w http.ResponseWriter, req *http.Request, model interface{}, _ string) error {
	w.Write([]byte(fmt.Sprintf("%s | %v", p.match, model)))
	return nil
}

func testLogger(t *testing.T) negotiator.Printer {
	return func(level byte, message string, data map[string]interface{}) {
		buf := &strings.Builder{}
		fmt.Fprintf(buf, "%c: %s", level, message)
		for k, v := range data {
			fmt.Fprintf(buf, ", %q: %v", k, v)
		}
		log.Printf(buf.String())
	}
}

type ValidXMLUser struct {
	Name string
}
