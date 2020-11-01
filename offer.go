package negotiator

const (
	Accept         = "Accept"
	AcceptLanguage = "Accept-Language"
	AcceptCharset  = "Accept-Charset"

	// AcceptEncoding is handled effectively by net/http and can be disregarded here

	XRequestedWith = "X-Requested-With"
	XMLHttpRequest = "XMLHttpRequest"
)

// BasicDataProvider is a function signature for obtaining data on request.
// The Data field in an Offer may be one of these functions.
// Note that its result can optionally be another BasicDataProvider or LanguageDataProvider.
type BasicDataProvider func() interface{}

// LanguageDataProvider is a function signature for obtaining data in a given language.
// The Data field in an Offer may be one of these functions.
// Note that its result can optionally be another BasicDataProvider or LanguageDataProvider.
type LanguageDataProvider func(language string) interface{}

// Offer holds the set of parameters that are offered to the content negotiation.
// Note that Data will be passed to a ResponseProcessor, having first checked whether
// it is a LanguageDataProvider, and if so that function will have been called with the
// chosen language as its parameter.
type Offer struct {
	MediaType string // e.g. "text/html" or blank not relevant
	Language  string // blank if not relevant
	Template  string // blank if not relevant
	Data      interface{}
}

// Offers is a slice of Offer.
type Offers []Offer

// MediaTypes gets the media types from the offers, keeping the same order.
func (offers Offers) MediaTypes() []string {
	ss := make([]string, len(offers))
	for i, o := range offers {
		ss[i] = o.MediaType
	}
	return ss
}

func (offers Offers) setDefaultWildcards() Offers {
	for _, o := range offers {
		// if any have blanks, update all that are blank
		if o.MediaType == "" || o.Language == "" {
			return offers.doSetDefaultWildcards()
		}
	}
	// no need to change anything
	return offers
}

func (offers Offers) doSetDefaultWildcards() Offers {
	ss := make(Offers, len(offers))
	for i, o := range offers {
		if o.MediaType == "" {
			o.MediaType = "*/*"
		}
		if o.Language == "" {
			o.Language = "*"
		}
		ss[i] = o
	}
	return ss
}

func dereferenceDataProviders(data interface{}, lang string) interface{} {
	for {
		switch fn := data.(type) {
		case BasicDataProvider:
			data = fn()
		case LanguageDataProvider:
			data = fn(lang)
		default:
			return data
		}
	}
}
