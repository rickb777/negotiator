package negotiator

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rickb777/negotiator/header"
	"github.com/rickb777/negotiator/processor"
)

const (
	xRequestedWith = "X-Requested-With"
	xmlHttpRequest = "XMLHttpRequest"
)

// ErrorHandler is called for NotAcceptable and InternalServerError situations.
type ErrorHandler func(w http.ResponseWriter, error string, code int)

// Printer is something that allows printing log entries. This is only used for diagnostics.
type Printer func(level byte, message string, data map[string]interface{})

// Negotiator is responsible for content negotiation when using custom response processors.
type Negotiator struct {
	processors   []processor.ResponseProcessor
	errorHandler ErrorHandler
	logger       Printer
}

// Default creates a negotiator with all the default processors, supporting
// JSON, XML, CSV and plain text.
func Default() *Negotiator {
	return New(processor.JSON(), processor.XML(), processor.CSV(), processor.TXT())
}

// New creates a Negotiator with a list of custom response processors.
func New(responseProcessors ...processor.ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   responseProcessors,
		errorHandler: http.Error,
		logger:       func(_ byte, _ string, _ map[string]interface{}) {},
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

// Insert more response processors. A new Negotiator is returned with the extra processors
// plus the original processors. The extra processors are inserted first.
// Because the processors are checked in order, any overlap of matching media range
// goes to the first such matching processor.
func (n *Negotiator) Insert(responseProcessors ...processor.ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   append(responseProcessors, n.processors...),
		errorHandler: n.errorHandler,
		logger:       n.logger,
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
		logger:       n.logger,
	}
}

// WithErrorHandler adds a custom error handler. This is used for 406-Not Acceptable cases
// and dealing with 500-Internal Server Error in Negotiate.
func (n *Negotiator) WithErrorHandler(eh ErrorHandler) *Negotiator {
	return &Negotiator{
		processors:   n.processors,
		errorHandler: eh,
		logger:       n.logger,
	}
}

// WithLogger adds a diagnostic logger.
func (n *Negotiator) WithLogger(printer Printer) *Negotiator {
	return &Negotiator{
		processors:   n.processors,
		errorHandler: n.errorHandler,
		logger:       printer,
	}
}

//-------------------------------------------------------------------------------------------------

// Negotiate negotiates your model based on the HTTP Accept and Accept-... headers.
// Any error arising will result in a 500 error response and a log message.
func (n *Negotiator) Negotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) {
	err := n.TryNegotiate(w, req, offers...)
	if err != nil {
		n.errorHandler(w, "the server was unable to complete this request", http.StatusInternalServerError)
		n.logger('E', "500: "+err.Error(),
			map[string]interface{}{
				"Accept":          req.Header.Get(Accept),
				"Accept-Language": req.Header.Get(AcceptLanguage),
				"Offers":          strings.Join(Offers(offers).MediaTypes(), ", "),
				"Error":           err,
			})
	}
}

// TryNegotiate your model based on the HTTP Accept and Accept-... headers.
// Usually, it will be sufficient to instead use Negotiate, which deals with error handling.
func (n *Negotiator) TryNegotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	r := n.Render(w, req, Offers(offers).setDefaultWildcards())
	r.WriteContentType(w)
	return r.Render(w)
}

// Render computes the best matching response, if there is one, and returns a suitable renderer
// that is compatible with Gin (github.com/gin-gonic/gin).
func (n *Negotiator) Render(w http.ResponseWriter, req *http.Request, offers Offers) Render {
	if IsAjax(req) {
		return n.ajaxNegotiate(offers)
	}

	if len(n.processors) == 0 {
		return unacceptable{n.errorHandler}
	}

	mrs := header.ParseMediaRanges(req.Header.Get(Accept)).WithDefault()
	languages := header.Parse(req.Header.Get(AcceptLanguage)).WithDefault()

	// first pass - remove offers that match exclusions
	// (this doesn't apply to language exclusions because we always allow at least one language match)
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

	// second pass - find the first matching media-range and language combination
	for i, offer := range offers {
		if !excluded[i] {
			offeredType, offeredSubtype := split(offer.MediaType, '/')

			for _, accepted := range mrs {
				for _, lang := range languages {
					n.info("200 compared", accepted.Value(), lang.Value, offer)

					if equalOrWildcard(accepted.Type, offeredType) &&
						equalOrWildcard(accepted.Subtype, offeredSubtype) &&
						equalOrWildcard(lang.Value, offer.Language) {

						if accepted.Quality > 0 && lang.Quality > 0 {
							for _, p := range n.processors {
								if p.CanProcess(offer.MediaType, offer.Language) {
									n.info("200 matched", accepted.Value(), lang.Value, offer)
									return process(p, offer)
								}
							}

							for _, p := range n.processors {
								if accepted.Type == "*" && accepted.Subtype == "*" {
									n.info("200 matched wildcard", accepted.Value(), lang.Value, offer)
									return process(p, offer)
								}
							}
						} else {
							// content matched but is explicitly excluded, so stop checking other matches
							return unacceptable{n.errorHandler}
						}
					}
				}
			}
		}
	}

	return unacceptable{n.errorHandler}
}

func process(p processor.ResponseProcessor, offer Offer) Render {
	return &renderer{
		data:     dereferenceDataProviders(offer.Data, offer.Language),
		language: offer.Language,
		template: offer.Template,
		p:        p,
	}
}

func (n *Negotiator) info(msg, accepted, lang string, offer Offer) {
	n.logger('D', msg,
		map[string]interface{}{
			"Accepted":   accepted,
			"Language":   lang,
			"OfferMedia": offer.MediaType,
			"OfferLang":  offer.Language,
		})
}

func (n *Negotiator) ajaxNegotiate(offers Offers) Render {
	for _, offer := range offers {
		if offer.MediaType == "*/*" || offer.MediaType == "application/*" || offer.MediaType == "application/json" {
			data := dereferenceDataProviders(offer.Data, "")

			for _, p := range n.processors {
				ajax, doesAjax := p.(processor.AjaxResponseProcessor)
				if doesAjax && ajax.IsAjaxResponder() {
					return &renderer{data: data, p: p}
				}
			}
		}
	}

	return unacceptable{n.errorHandler}
}

// IsAjax tests whether a request has the Ajax header sent by browsers for XHR requests.
func IsAjax(req *http.Request) bool {
	return req.Header.Get(XRequestedWith) == XMLHttpRequest
}

func equalOrWildcard(accepted, offered string) bool {
	return offered == "*" || accepted == "*" || accepted == offered
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
