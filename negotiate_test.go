package negotiator_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/onsi/gomega"
	"github.com/rickb777/negotiator"
	"github.com/rickb777/negotiator/processor"
)

type User struct {
	Name string
}

// Negotiate applies the negotiation algorithm, choosing the response
// based on the Accept header in the request, if present.
// It returns either a successful response or a 406-Not Acceptable,
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
		negotiator.New().WithDefaults().Negotiate(w, req, negotiator.Offer{Data: user})
	}

	// normal handling
	http.Handle("/user", http.HandlerFunc(getUser))
}

// Negotiate applies the negotiation algorithm, choosing the response
// based on the Accept header in the request, if present.
// It returns either a successful response or a 406-Not Acceptable.
//
// In this example, there is only one offer and it will be used by whichever
// response processor matches the request. The example integrates the negotiator
// seamlessly with Gin using the Context.Render method.
func ExampleNegotiator_Render_singleOffer() {
	// create and configure Gin engine, e.g.
	engine := gin.Default()

	// getUser is a 'standard' handler function
	getUser := func(c *gin.Context) {
		// some data; this will be wrapped in an Offer{}
		user := &User{Name: "Joe Bloggs"}

		// the negotiator determines the response format based on the request headers
		// returning a CodedRender value
		cr := negotiator.New().WithDefaults().Render(c.Request, negotiator.Offer{Data: user})

		// pass the negotiation result to Gin; the status code will be one of
		// 200-OK, 204-No content, or 406-Not acceptable
		c.Render(cr.StatusCode(), cr)
	}

	// normal handling
	engine.GET("/user", getUser)
}

func Test_should_add_custom_response_processors(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New().Append(processor.JSON(), processor.XML()).Append(fakeResponseProcessor)

	lastProcessor := n.Processor(n.N() - 1)
	processorName := reflect.TypeOf(lastProcessor).String()

	g.Expect(n.N()).To(gomega.Equal(3))
	g.Expect(processorName).To(gomega.Equal("*negotiator_test.fakeProcessor"))
}

func Test_should_add_custom_response_processors_to_end(t *testing.T) {
	g := gomega.NewWithT(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New().WithDefaults().Append(fakeResponseProcessor)

	lastProcessor := n.Processor(n.N() - 1)
	processorName := reflect.TypeOf(lastProcessor).String()

	g.Expect(n.N()).To(gomega.Equal(5))
	g.Expect(processorName).To(gomega.Equal("*negotiator_test.fakeProcessor"))
}

//-------------------------------------------------------------------------------------------------

func Test_should_unpack_lazy_data(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var a = &fakeProcessor{match: "text/html"}
	n := negotiator.New(a)

	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	fn2 := func(lang string) interface{} {
		return lang
	}
	fn1 := func() interface{} {
		return fn2
	}
	n.Negotiate(recorder, req, negotiator.Offer{Data: fn1, Language: "en"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/html | en"))
}

func Test_should_use_default_processor_if_no_accept_header(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var a = &fakeProcessor{match: "text/test"}
	var b = &fakeProcessor{match: "text/plain"}
	n := negotiator.New(a, b)

	req, _ := http.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_give_JSON_response_for_ajax_requests(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	n := negotiator.New().WithDefaults()

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(negotiator.XRequestedWith, negotiator.XMLHttpRequest)
	recorder := httptest.NewRecorder()

	model := &ValidXMLUser{Name: "Joe Bloggs"}
	n.Negotiate(recorder, req, negotiator.Offer{Data: model})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("{\"Name\":\"Joe Bloggs\"}\n"))
}

func Test_should_give_406_for_unmatched_ajax_requests(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	n := negotiator.New().WithDefaults()

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(negotiator.XRequestedWith, negotiator.XMLHttpRequest)
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/plain"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusNotAcceptable))
}

func Test_should_return_406_if_no_matching_accept_header(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	cases := []string{"application/xml", "text/test"}

	for _, c := range cases {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Add("Accept", "image/png")
		recorder := httptest.NewRecorder()

		n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: c})

		g.Expect(recorder.Code).To(gomega.Equal(http.StatusNotAcceptable))
	}
}

// RFC7231 suggests that 406 is sent when no media range matches are possible.
func Test_should_return_406_when_media_range_is_explicitly_excluded(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but text/test"
	req.Header.Add("Accept", "text/test;q=0, */*") // excluded
	req.Header.Add("Accept-Language", "en")        // accepted
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test", Language: "en"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusNotAcceptable))
}

// RFC7231 recommends that, when no language matches are possible, a response should be sent anyway.
func Test_should_return_200_even_when_language_is_explicitly_excluded(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	// this header means "anything but text/test"
	req.Header.Add("Accept", "text/test, */*")
	req.Header.Add("Accept-Language", "en;q=0 *") // anything but "en"
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test", Language: "en"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
}

func Test_should_negotiate_and_write_to_response_body(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var p1 = &fakeProcessor{match: "text/html"}
	var p2 = &fakeProcessor{match: "text/test"}
	n := negotiator.New(p1, p2)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/test, text/*")
	req.Header.Add("Accept-Language", "en-GB, fr-FR")
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req,
		// should be skipped because of media mismatch
		negotiator.Offer{Data: "d1", MediaType: "text/html", Language: "en"},
		// should be skipped because of language mismatch
		negotiator.Offer{Data: "d2", MediaType: "text/test", Language: "de"},
		// should match
		negotiator.Offer{Data: "d3", MediaType: "text/test", Language: "en"},
	)

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | d3"))
}

func Test_should_match_subtype_wildcard(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/*")
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/test"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_match_language_when_offer_language_is_not_specified(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/html"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/html")
	req.Header.Add("Accept-Language", "en, fr")
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/html"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/html | foo"))
}

func Test_should_match_language_wildcard_and_send_content_language_header(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept-Language", "*")
	recorder := httptest.NewRecorder()

	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", Language: "en"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Header().Get("Content-Language")).To(gomega.Equal("en"))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))
}

func Test_should_negotiate_a_default_processor(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var fakeResponseProcessor = &fakeProcessor{match: "text/test"}
	n := negotiator.New(fakeResponseProcessor)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "*/*")

	recorder := httptest.NewRecorder()
	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | foo"))

	recorder = httptest.NewRecorder()
	n.Negotiate(recorder, req, negotiator.Offer{Data: "bar", MediaType: "text/test"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/test | bar"))
}

func Test_should_negotiate_one_of_the_processors(t *testing.T) {
	g := gomega.NewWithT(t)
	testLogger(t)
	var a = &fakeProcessor{match: "text/a"}
	var b = &fakeProcessor{match: "text/b"}
	n := negotiator.New(a, b)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/a, text/b")

	recorder := httptest.NewRecorder()
	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/a"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/a | foo"))

	recorder = httptest.NewRecorder()
	n.Negotiate(recorder, req, negotiator.Offer{Data: "bar", MediaType: "text/b"})

	g.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
	g.Expect(recorder.Body.String()).To(gomega.Equal("text/b | bar"))
}

func Test_should_negotiate_and_panic_on_error(t *testing.T) {
	testLogger(t)
	var a = &fakeProcessor{match: "text/a", err: errors.New("ouch!")}
	n := negotiator.New(a)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/a")

	defer func() {
		recover()
	}()

	recorder := httptest.NewRecorder()
	n.Negotiate(recorder, req, negotiator.Offer{Data: "foo", MediaType: "text/a"})

	t.Error("should not reach here")
}

//-------------------------------------------------------------------------------------------------

type fakeProcessor struct {
	match string
	err   error
}

func (p *fakeProcessor) ContentType() string {
	return p.match
}

func (p *fakeProcessor) CanProcess(mediaRange string, lang string) bool {
	return mediaRange == p.match && (lang == "*" || lang == "en")
}

func (p *fakeProcessor) Process(w http.ResponseWriter, _ string, data interface{}) {
	if p.err != nil {
		panic(p.err)
	}
	w.Write([]byte(fmt.Sprintf("%s | %v", p.match, data)))
}

func testLogger(t *testing.T) {
	negotiator.Printer = func(level byte, message string, data map[string]interface{}) {
		buf := &strings.Builder{}
		fmt.Fprintf(buf, "%c: %s", level, message)
		for k, v := range data {
			fmt.Fprintf(buf, ", %q: %v", k, v)
		}
		t.Logf(buf.String())
	}
}

type ValidXMLUser struct {
	Name string
}
