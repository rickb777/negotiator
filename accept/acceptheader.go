package accept

import (
	"sort"
	"strconv"
	"strings"
)

const (
	// DefaultQuality is the default weight of a media range without explicit "q"
	// https://tools.ietf.org/html/rfc7231#section-5.3.1
	DefaultQuality float64 = 1.0 //e.g text/html;q=1
)

// ParseAcceptHeader splits a prioritised Accept header value and sorts the parts.
// These are returned in order with the most preferred first.
//
// A request without any Accept header field implies that the user agent
// will accept any media type in response.  If the header field is
// present in a request and none of the available representations for
// the response have a media type that is listed as acceptable, the
// origin server can either honor the header field by sending a 406 (Not
// Acceptable) response or disregard the header field by treating the
// response as if it is not subject to content negotiation.
func ParseAcceptHeader(acceptHeader string) []MediaRange {
	wvs := parseHeader(acceptHeader)
	result := splitMediaRanges(wvs)
	sort.Stable(mrByPrecedence(result))
	return result
}

// ParseAcceptLanguageHeader splits a prioritised Accept-Language header value and
// sorts the parts. These are returned in order with the most preferred first.
func ParseAcceptLanguageHeader(acceptHeader string) []PrecedenceValue {
	wvs := parseHeader(acceptHeader)
	sort.Stable(wvByPrecedence(wvs))
	return wvs
}

// ParseAcceptEncodingHeader splits a prioritised Accept-Encoding header value and
// sorts the parts. These are returned in order with the most preferred first.
func ParseAcceptEncodingHeader(acceptHeader string) []PrecedenceValue {
	wvs := parseHeader(acceptHeader)
	sort.Stable(wvByPrecedence(wvs))
	return wvs
}

// ParseAcceptCharsetHeader splits a prioritised Accept-Charset header value and
// sorts the parts. These are returned in order with the most preferred first.
func ParseAcceptCharsetHeader(acceptHeader string) []PrecedenceValue {
	wvs := parseHeader(acceptHeader)
	sort.Stable(wvByPrecedence(wvs))
	return wvs
}

func parseHeader(acceptHeader string) []PrecedenceValue {
	if acceptHeader == "" {
		return nil
	}

	parts := strings.Split(acceptHeader, ",")
	wvs := make([]PrecedenceValue, 0, len(parts))

	for _, part := range parts {
		mrAndAcceptParam := strings.Split(part, ";")
		//if no accept-param
		if len(mrAndAcceptParam) == 1 {
			wvs = append(wvs, handlePartWithoutParams(mrAndAcceptParam[0]))
		} else {
			wvs = append(wvs, handlePartWithParams(mrAndAcceptParam[0], mrAndAcceptParam[1:]))
		}
	}

	return wvs
}

func splitMediaRanges(wvs []PrecedenceValue) []MediaRange {
	if wvs == nil {
		return nil
	}

	mrs := make([]MediaRange, len(wvs))
	for i, wv := range wvs {
		t, s := split(wv.Value, '/')
		mrs[i] = MediaRange{
			Type:    t,
			Subtype: s,
			Quality: wv.Quality,
			Params:  wv.Params,
		}
	}
	return mrs
}

func handlePartWithParams(value string, acceptParams []string) PrecedenceValue {
	wv := new(PrecedenceValue)
	wv.Value = strings.TrimSpace(value)
	wv.Quality = DefaultQuality

	for _, ap := range acceptParams {
		ap = strings.TrimSpace(ap)
		k, v := split(ap, '=')
		if strings.TrimSpace(k) == "q" {
			wv.Quality = parseQuality(v)
		} else {
			wv.Params = append(wv.Params, KV{Key: k, Value: v})
		}
	}
	return *wv
}

func isQualityAcceptParam(acceptParam string) bool {
	return strings.HasPrefix(acceptParam, "q=")
}

func parseQuality(qstring string) float64 {
	weight, err := strconv.ParseFloat(qstring, 64)
	if err != nil {
		weight = 1.0
	}
	return weight
}

func handlePartWithoutParams(value string) PrecedenceValue {
	return PrecedenceValue{
		Value:   strings.TrimSpace(value),
		Quality: DefaultQuality,
	}
}

func split(value string, b byte) (string, string) {
	slash := strings.IndexByte(value, b)
	if slash < 0 {
		return value, ""
	}
	return value[:slash], value[slash+1:]
}
