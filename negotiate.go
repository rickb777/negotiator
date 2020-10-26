package negotiator

import (
	"fmt"
	"github.com/rickb777/negotiator/header"
	"log"
	"net/http"
	"strings"
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

// NewWithJSONAndXML allows users to pass custom response processors. By default, processors
// for XML and JSON are already created.
func NewWithJSONAndXML(responseProcessors ...ResponseProcessor) *Negotiator {
	return New(append(responseProcessors, NewJSON(), NewXML())...)
}

//New allows users to pass custom response processors.
func New(responseProcessors ...ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   responseProcessors,
		errorHandler: http.Error,
		logger:       func(_ byte, _ string, _ map[string]interface{}) {},
	}
}

// Add more response processors. A new Negotiator is returned with the original processors plus
// the extra processors.
func (n *Negotiator) Add(responseProcessors ...ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   append(n.processors, responseProcessors...),
		errorHandler: n.errorHandler,
		logger:       func(_ byte, _ string, _ map[string]interface{}) {},
	}
}

// With adds a custom error handler. This is used for 406-Not Acceptable cases.
func (n *Negotiator) With(eh ErrorHandler) *Negotiator {
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

// NegotiateWithJSONAndXML handles your model based on the HTTP Accept-Xyz headers. Only XML and JSON are handled.
func NegotiateWithJSONAndXML(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	return NewWithJSONAndXML().Negotiate(w, req, offers...)
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

	mrs := header.ParseMediaRanges(req.Header.Get(Accept)).WithDefault()
	languages := header.Parse(req.Header.Get(AcceptLanguage)).WithDefault()
	//charsets := header.Parse(req.Header.Get(AcceptCharset)).WithDefault()
	//encodings := header.Parse(req.Header.Get(AcceptEncoding)).WithDefault()

	// The first pass checks for exact matches, and this includes exclusions (q=0)
	// The second pass allows wildcard matching.
	for pass := 1; pass <= 2; pass++ {
		allowWildcard := pass == 2

		for _, offer := range offers {
			offeredType, offeredSubtype := split(offer.MediaType, '/')

			for _, accepted := range mrs {
				for _, lang := range languages {
					n.info("200 compared", accepted.Value(), lang.Value, offer)

					if equalOrWildcard(accepted.Type, offeredType, allowWildcard) &&
						equalOrWildcard(accepted.Subtype, offeredSubtype, allowWildcard) &&
						equalOrWildcard(lang.Value, offer.Language, allowWildcard) {

						if accepted.Quality > 0 && lang.Quality > 0 {
							for _, processor := range n.processors {
								data := dereferenceDataProviders(offer.Data, offer.Language)
								if accepted.Type == "*" && accepted.Subtype == "*" {
									n.info("200 matched wildcard", accepted.Value(), lang.Value, offer)
									return processor.Process(w, req, data, offer.Template)
								} else if processor.CanProcess(offer.MediaType, offer.Language) {
									n.info("200 matched", accepted.Value(), lang.Value, offer)
									return processor.Process(w, req, data, offer.Template)
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

func equalOrWildcard(accepted, offered string, allowWildcard bool) bool {
	return offered == "" || accepted == offered ||
		(allowWildcard && accepted == "*")
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
