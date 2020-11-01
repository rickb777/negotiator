package negotiator

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rickb777/negotiator/header"
)

const (
	xRequestedWith = "X-Requested-With"
	xmlHttpRequest = "XMLHttpRequest"
)

// ErrorHandler is called for NotAcceptable situations.
type ErrorHandler func(w http.ResponseWriter, error string, code int)

// Printer is something that allows printing log entries. This is only used for diagnostics.
type Printer func(level byte, message string, data map[string]interface{})

// Negotiator is responsible for content negotiation when using custom response processors.
type Negotiator struct {
	processors   []ResponseProcessor
	errorHandler ErrorHandler
	logger       Printer
}

// Default creates a negotiator with all the default processors, supporting
// JSON, XML, CSV and plain text.
func Default() *Negotiator {
	return New(JSONProcessor(), XMLProcessor(), CSVProcessor(), TXTProcessor())
}

// New creates a Negotiator with a list of custom response processors.
func New(responseProcessors ...ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   responseProcessors,
		errorHandler: http.Error,
		logger:       func(_ byte, _ string, _ map[string]interface{}) {},
	}
}

// Processor gets the ith processor.
func (n *Negotiator) Processor(i int) ResponseProcessor {
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
func (n *Negotiator) Insert(responseProcessors ...ResponseProcessor) *Negotiator {
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
func (n *Negotiator) Append(responseProcessors ...ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   append(n.processors, responseProcessors...),
		errorHandler: n.errorHandler,
		logger:       n.logger,
	}
}

// WithErrorHandler adds a custom error handler. This is used for 406-Not Acceptable cases
// and dealing with 500-Internal Server Error in MustNegotiate.
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

// MustNegotiate negotiates your model based on the HTTP Accept and Accept-... headers.
// Any error arising will result in a 500 error response and a log message.
func (n *Negotiator) MustNegotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) {
	err := n.Negotiate(w, req, offers...)
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

// Negotiate your model based on the HTTP Accept and Accept-... headers.
func (n *Negotiator) Negotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	if IsAjax(req) {
		return n.ajaxNegotiate(w, req, offers...)
	}

	if len(n.processors) == 0 {
		return n.notAcceptable(w)
	}

	return n.negotiate(w, req, Offers(offers).setDefaultWildcards())
}

func (n *Negotiator) negotiate(w http.ResponseWriter, req *http.Request, offers Offers) error {
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
							for _, processor := range n.processors {
								if processor.CanProcess(offer.MediaType, offer.Language) {
									n.info("200 matched", accepted.Value(), lang.Value, offer)
									return process(processor, w, req, offer)
								}
							}

							for _, processor := range n.processors {
								if accepted.Type == "*" && accepted.Subtype == "*" {
									n.info("200 matched wildcard", accepted.Value(), lang.Value, offer)
									return process(processor, w, req, offer)
								}
							}
						} else {
							// content matched but is explicitly excluded, so stop checking other matches
							return n.notAcceptable(w)
						}
					}
				}
			}
		}
	}

	return n.notAcceptable(w)
}

func process(processor ResponseProcessor, w http.ResponseWriter, req *http.Request, offer Offer) error {
	if offer.Language != "" {
		w.Header().Set("Content-Language", offer.Language)
	}
	data := dereferenceDataProviders(offer.Data, offer.Language)
	return processor.Process(w, req, data, offer.Template)
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

func (n *Negotiator) ajaxNegotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	for _, offer := range offers {
		if offer.MediaType == "" || offer.MediaType == "application/json" {
			data := dereferenceDataProviders(offer.Data, "")

			for _, processor := range n.processors {
				ajax, doesAjax := processor.(AjaxResponseProcessor)
				if doesAjax && ajax.IsAjaxResponder() {
					return processor.Process(w, req, data, "")
				}
			}
		}
	}

	return n.notAcceptable(w)
}

// IsAjax tests whether a request has the Ajax header sent by browsers for XHR requests.
func IsAjax(req *http.Request) bool {
	return req.Header.Get(XRequestedWith) == XMLHttpRequest
}

func (n *Negotiator) notAcceptable(w http.ResponseWriter) error {
	n.errorHandler(w, "the accepted formats are not offered by the server", http.StatusNotAcceptable)
	return nil
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
