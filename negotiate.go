package negotiator

import (
	"github.com/rickb777/negotiator/header"
	"net/http"
	"strings"
)

const (
	xRequestedWith = "X-Requested-With"
	xmlHttpRequest = "XMLHttpRequest"
)

// ErrorHandler is called for NotAcceptable situations.
type ErrorHandler func(w http.ResponseWriter, error string, code int)

// Negotiator is responsible for content negotiation when using custom response processors.
type Negotiator struct {
	processors   []ResponseProcessor
	errorHandler ErrorHandler
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
	}
}

// Add more response processors. A new Negotiator is returned with the original processors plus
// the extra processors.
func (n *Negotiator) Add(responseProcessors ...ResponseProcessor) *Negotiator {
	return &Negotiator{
		processors:   append(n.processors, responseProcessors...),
		errorHandler: n.errorHandler,
	}
}

// With adds a custom error handler. This is used for 406-Not Acceptable cases.
func (n *Negotiator) With(eh ErrorHandler) *Negotiator {
	return &Negotiator{
		processors:   n.processors,
		errorHandler: eh,
	}
}

// NegotiateWithJSONAndXML handles your model based on the HTTP Accept-Xyz headers. Only XML and JSON are handled.
func NegotiateWithJSONAndXML(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	return NewWithJSONAndXML().Negotiate(w, req, offers...)
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

	for _, accepted := range mrs {
		for _, lang := range languages {
			for _, offer := range offers {
				t, s := split(offer.MediaType, '/')
				if equalOrWildcard(accepted.Type, t) &&
					equalOrWildcard(accepted.Subtype, s) &&
					equalOrWildcard(lang.Value, offer.Language) {

					if accepted.Quality > 0 && lang.Quality > 0 {
						for _, processor := range n.processors {
							data := dereferenceDataProviders(offer.Data, offer.Language)
							if accepted.Type == "*" && accepted.Subtype == "*" {
								return processor.Process(w, req, data, offer.Template)
							} else if processor.CanProcess(offer.MediaType, offer.Language) {
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

	return n.notAcceptable(w)
}

func (n *Negotiator) ajaxNegotiate(w http.ResponseWriter, req *http.Request, offers ...Offer) error {
	for _, offer := range offers {
		if offer.MediaType == "application/json" {
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

// Firstly, all Ajax requests are processed by the first available Ajax processor.
// Otherwise, standard content negotiation kicks in.
//
// A request without any Accept header field implies that the user agent
// will accept any media type in response.
//
// If the header field is present in a request and none of the available
// representations for the response have a media type that is listed as
// acceptable, the origin server can either honour the header field by
// sending a 406 (Not Acceptable) response or disregard the header field
// by treating the response as if it is not subject to content negotiation.
// This implementation prefers the former.
//
// See rfc7231-sec5.3.2:
// http://tools.ietf.org/html/rfc7231#section-5.3.2
//func (n *Negotiator) negotiateHeader(w http.ResponseWriter, req *http.Request, dataModel interface{}, template string) error {
//	if fn, ok := dataModel.(func() interface{}); ok {
//		dataModel = fn()
//	}
//
//	if IsAjax(req) {
//		for _, processor := range n.processors {
//			ajax, doesAjax := processor.(AjaxResponseProcessor)
//			if doesAjax && ajax.IsAjaxResponder() {
//				return processor.Process(w, req, dataModel, template)
//			}
//		}
//	}
//
//	if len(n.processors) > 0 {
//		mrs := header.ParseMediaRanges(req.Header.Get(Accept)).WithDefault()
//		charsets := header.Parse(req.Header.Get(AcceptCharset)).WithDefault()
//		languages := header.Parse(req.Header.Get(AcceptLanguage)).WithDefault()
//
//		for _, accepted := range mrs {
//			if accepted.Quality > 0 {
//				for _, cs := range charsets {
//					if cs.Quality > 0 {
//						for _, lang := range languages {
//							if lang.Quality > 0 {
//								for _, processor := range n.processors {
//									if processor.CanProcess(accepted.Value(), lang.Value) {
//										return processor.Process(w, req, dataModel, template)
//									}
//								}
//							}
//						}
//					}
//				}
//			}
//		}
//	}
//
//	n.errorHandler(w, "", http.StatusNotAcceptable)
//	return nil
//}

// IsAjax tests whether a request has the Ajax header sent by browsers for XHR requests.
func IsAjax(req *http.Request) bool {
	return req.Header.Get(XRequestedWith) == XMLHttpRequest
}

func (n *Negotiator) notAcceptable(w http.ResponseWriter) error {
	n.errorHandler(w, "the accepted formats are not offered by the server", http.StatusNotAcceptable)
	return nil
}

func equalOrWildcard(accepted, offered string) bool {
	return offered == "" || accepted == "*" || accepted == offered
}

func split(value string, b byte) (string, string) {
	i := strings.IndexByte(value, b)
	if i < 0 {
		return value, ""
	}
	return value[:i], value[i+1:]
}
