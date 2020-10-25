package negotiator

const (
	Accept         = "Accept"
	AcceptLanguage = "Accept-Language"
	AcceptCharset  = "Accept-Charset"
	AcceptEncoding = "Accept-Encoding"

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
	MediaType string
	Language  string // blank if not relevant
	Template  string // blank if not relevatn
	Data      interface{}
}

type Offers []Offer

func (offers Offers) MediaTypes() []string {
	ss := make([]string, len(offers))
	for i, o := range offers {
		ss[i] = o.MediaType
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
