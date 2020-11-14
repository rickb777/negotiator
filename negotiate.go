package negotiator

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rickb777/negotiator/header"
	"github.com/rickb777/negotiator/processor"
)

//const (
//	xRequestedWith = "X-Requested-With"
//	xmlHttpRequest = "XMLHttpRequest"
//)

// ErrorHandler is called for NotAcceptable and InternalServerError situations.
type ErrorHandler func(w http.ResponseWriter, error string, code int)

// Printer is something that allows printing log entries. This is only used for diagnostics.
var Printer = func(level byte, message string, data map[string]interface{}) {}

// Negotiator is responsible for content negotiation when using custom response processors.
type Negotiator struct {
	processors   []processor.ResponseProcessor
	errorHandler ErrorHandler
}

// New creates a Negotiator with a list of custom response processors. The error handler
// invokes http.Error and the diagnostic printer is no-op; change these if required.
func New(responseProcessors ...processor.ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   responseProcessors,
		errorHandler: http.Error,
	}
}

// Append more response processors. A new Negotiator is returned with the original processors
// plus the extra processors. The extra processors are appended last.
// Because the processors are checked in order, any overlap of matching media range
// goes to the first such matching processor.
func (n *Negotiator) Append(responseProcessors ...processor.ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   append(n.processors, responseProcessors...),
		errorHandler: n.errorHandler,
	}
}

// WithDefaults adds the default processors JSON, XML, CSV and TXT.
func (n *Negotiator) WithDefaults() *Negotiator {
	return &Negotiator{
		processors:   append(n.processors, processor.JSON(), processor.XML(), processor.CSV(), processor.TXT()),
		errorHandler: n.errorHandler,
	}
}

// WithErrorHandler adds a custom error handler. This is used for 406-Not Acceptable cases
// and dealing with 500-Internal Server Error in Negotiate.
func (n *Negotiator) WithErrorHandler(eh ErrorHandler) *Negotiator {
	return &Negotiator{
		processors:   n.processors,
		errorHandler: eh,
	}
}

// Processor gets the ith processor.
func (n *Negotiator) Processor(i int) processor.ResponseProcessor {
	return n.processors[i]
}

// N returns the number of processors.
func (n *Negotiator) N() int {
	return len(n.processors)
}

//-------------------------------------------------------------------------------------------------

// Negotiate negotiates your model based on the HTTP Accept and Accept-... headers.
// Any error arising will result in a panic.
func (n *Negotiator) Negotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) {
	r := n.Render(req, offers...)
	r.WriteContentType(w)
	w.WriteHeader(r.StatusCode())
	err := r.Render(w)
	if err != nil {
		panic(fmt.Errorf("%s %s %w", req.Method, req.URL, err))
	}
}

// Render computes the best matching response, if there is one, and returns a suitable renderer
// that is compatible with Gin (github.com/gin-gonic/gin).
func (n *Negotiator) Render(req *http.Request, offers ...Offer) CodedRender {
	offers = Offers(offers).setDefaultWildcards()

	if IsAjax(req) {
		return n.ajaxNegotiate(offers)
	}

	mrs := header.ParseMediaRanges(req.Header.Get(Accept)).WithDefault()
	languages := header.Parse(req.Header.Get(AcceptLanguage)).WithDefault()

	if len(n.processors) == 0 {
		info2("406 no processors configured", "Accept", mrs.String(), "Accept-Language", languages.String())
		return unacceptable{n.errorHandler}
	}

	// first pass - remove offers that match exclusions
	// (this doesn't apply to language exclusions because we always allow at least one language match)
	remaining := removeExcludedOffers(offers, mrs)

	// second pass - find the first exact-match media-range and language combination
	for _, offer := range remaining {
		p := n.findBestMatch(mrs, languages, offer, exactMatch)
		if p != nil {
			return process(p, offer)
		}
	}

	// third pass - find the first near-match media-range and language combination
	for _, offer := range remaining {
		p := n.findBestMatch(mrs, languages, offer, nearMatch)
		if p != nil {
			return process(p, offer)
		}
	}

	info2("406 rejected", "Accept", mrs.String(), "Accept-Language", languages.String())
	return unacceptable{n.errorHandler}
}

func (n *Negotiator) findBestMatch(mrs header.MediaRanges, languages header.PrecedenceValues, offer Offer,
	match func(header.MediaRange, header.PrecedenceValue, Offer) bool) processor.ResponseProcessor {

	for _, accepted := range mrs {
		for _, lang := range languages {
			info("compared", accepted.Value(), lang.Value, offer)

			if match(accepted, lang, offer) {
				if lang.Quality > 0 {
					if offer.MediaType == "*/*" {
						// default to the first processor
						info("200 matched wildcard", accepted.Value(), lang.Value, offer)
						return n.processors[0]
					}

					// find the first matching processor
					for _, p := range n.processors {
						if p.CanProcess(offer.MediaType, offer.Language) {
							info("200 matched", accepted.Value(), lang.Value, offer)
							return p
						}
					}
				}
			}
		}
	}

	return nil
}

// Any media range
func removeExcludedOffers(offers Offers, mrs header.MediaRanges) Offers {
	excluded := make([]bool, len(offers))
	for i, offer := range offers {
		offeredType, offeredSubtype := split(offer.MediaType, '/')

		for _, accepted := range mrs {
			if accepted.Quality <= 0 &&
				accepted.Type == offeredType &&
				accepted.Subtype == offeredSubtype {

				excluded[i] = true
			}
		}
	}

	remaining := make(Offers, 0, len(offers))
	for i, offer := range offers {
		if !excluded[i] {
			remaining = append(remaining, offer)
		}
	}

	return remaining
}

func exactMatch(accepted header.MediaRange, lang header.PrecedenceValue, offer Offer) bool {
	offeredType, offeredSubtype := split(offer.MediaType, '/')
	return accepted.Type == offeredType &&
		accepted.Subtype == offeredSubtype &&
		equalOrPrefix(lang.Value, offer.Language)
}

func nearMatch(accepted header.MediaRange, lang header.PrecedenceValue, offer Offer) bool {
	offeredType, offeredSubtype := split(offer.MediaType, '/')
	return equalOrWildcard(accepted.Type, offeredType) &&
		equalOrWildcard(accepted.Subtype, offeredSubtype) &&
		equalOrPrefix(lang.Value, offer.Language)
}

func equalOrPrefix(acceptedLang, offeredLang string) bool {
	return acceptedLang == "*" ||
		offeredLang == "*" ||
		acceptedLang == offeredLang ||
		strings.HasPrefix(acceptedLang, offeredLang+"-")
}

func equalOrWildcard(accepted, offered string) bool {
	return offered == "*" || accepted == "*" || accepted == offered
}

//-------------------------------------------------------------------------------------------------

func process(p processor.ResponseProcessor, offer Offer) CodedRender {
	data := dereferenceDataProviders(offer.Data, offer.Language)
	if data == nil {
		return emptyCode(http.StatusNoContent)
	}

	return &renderer{
		data:        data,
		language:    offer.Language,
		template:    offer.Template,
		contentType: p.ContentType(),
		process:     p.Process,
	}
}

func info(msg, accepted, lang string, offer Offer) {
	info2(msg,
		"Accepted", accepted,
		"Language", lang,
		"OfferMedia", offer.MediaType,
		"OfferLang", offer.Language)
}

func info2(msg string, vv ...interface{}) {
	m := make(map[string]interface{})
	var s string
	for i := 1; i < len(vv); i += 2 {
		s = vv[i-1].(string)
		m[s] = vv[i]
	}
	Printer('D', msg, m)
}

func (n *Negotiator) ajaxNegotiate(offers Offers) CodedRender {
	for _, offer := range offers {
		if offer.MediaType == "*/*" || offer.MediaType == "application/*" || offer.MediaType == "application/json" {
			data := dereferenceDataProviders(offer.Data, offer.Language)
			return &renderer{
				data:        data,
				language:    offer.Language,
				contentType: "application/json; charset=utf-8",
				process:     processor.RenderJSON(""),
			}
		}
	}

	return unacceptable{n.errorHandler}
}

// IsAjax tests whether a request has the Ajax header sent by browsers for XHR requests.
func IsAjax(req *http.Request) bool {
	return req.Header.Get(XRequestedWith) == XMLHttpRequest
}

func split(value string, b byte) (string, string) {
	i := strings.IndexByte(value, b)
	if i < 0 {
		return value, ""
	}
	return value[:i], value[i+1:]
}

// StdLogger adapts the standard Go logger to be usable for the negotiator.
var StdLogger = func(level byte, message string, data map[string]interface{}) {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "%c: %s", level, message)
	for k, v := range data {
		fmt.Fprintf(buf, ", %q: %v", k, v)
	}
	log.Printf(buf.String())
}
