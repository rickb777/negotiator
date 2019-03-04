package negotiator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcceptParse_parses_empty(t *testing.T) {
	a := Accept("")
	mr := a.Parse()

	assert.Equal(t, 0, len(mr))
}

func TestAcceptParse_parses_single(t *testing.T) {
	a := Accept("application/json")
	mr := a.Parse()

	assert.Equal(t, 1, len(mr))
	assert.Equal(t, "application/json", mr[0].Value)
	assert.Equal(t, TypeSubtypeMediaRangeWeight, mr[0].Weight)
}

func TestAcceptParse_preserves_case_of_mediaRange(t *testing.T) {
	a := Accept("application/CEA")
	mr := a.Parse()

	assert.Equal(t, 1, len(mr))
	assert.Equal(t, "application/CEA", mr[0].Value)
}

func TestAcceptParse_defaults_quality_if_not_explicit(t *testing.T) {
	a := Accept("text/plain")
	mr := a.Parse()
	assert.Equal(t, 1, len(mr))
	assert.Equal(t, TypeSubtypeMediaRangeWeight, mr[0].Weight)
}

func TestAcceptParse_should_parse_quality(t *testing.T) {
	a := Accept("application/json;q=0.9")
	mr := a.Parse()

	assert.Equal(t, 1, len(mr))
	assert.Equal(t, "application/json", mr[0].Value)
	assert.Equal(t, 0.9, mr[0].Weight)
}

func TestAcceptParse_should_parse_multi_qualities(t *testing.T) {
	a := Accept("application/xml;q=1, application/json;q=0.9")
	mr := a.Parse()

	assert.Equal(t, 2, len(mr))

	assert.Equal(t, "application/xml", mr[0].Value)
	assert.Equal(t, 1.0, mr[0].Weight)

	assert.Equal(t, "application/json", mr[1].Value)
	assert.Equal(t, 0.9, mr[1].Weight)
}

func TestAcceptParse_should_also_handle_languages(t *testing.T) {
	a := Accept("en-GB,en;q=0.5")
	langs := a.Parse()

	assert.Equal(t, 2, len(langs))

	assert.Equal(t, "en-GB", langs[0].Value)
	assert.Equal(t, 1.0, langs[0].Weight)

	assert.Equal(t, "en", langs[1].Value)
	assert.Equal(t, 0.5, langs[1].Weight)
}

func TestAcceptParse_should_also_handle_encoding(t *testing.T) {
	a := Accept("compress;q=0.5, gzip;q=1.0")
	langs := a.Parse().Values()

	assert.Equal(t, 2, len(langs))
	assert.Equal(t, "gzip", langs[0])
	assert.Equal(t, "compress", langs[1])
}

func TestAcceptParse_should_also_handle_charsets(t *testing.T) {
	a := Accept("iso-8859-5, unicode-1-1;q=0.8")
	langs := a.Parse()

	assert.Equal(t, 2, len(langs))

	assert.Equal(t, "iso-8859-5", langs[0].Value)
	assert.Equal(t, 1.0, langs[0].Weight)

	assert.Equal(t, "unicode-1-1", langs[1].Value)
	assert.Equal(t, 0.8, langs[1].Weight)
}

func TestAcceptParse_reorders_by_quality_decending(t *testing.T) {
	a := Accept("application/json;q=0.8, application/xml")
	mr := a.Parse()

	assert.Equal(t, 2, len(mr))

	assert.Equal(t, "application/xml", mr[0].Value)
	assert.Equal(t, TypeSubtypeMediaRangeWeight, mr[0].Weight)

	assert.Equal(t, "application/json", mr[1].Value)
	assert.Equal(t, 0.8, mr[1].Weight)
}

func TestMediaRanges_should_ignore_invalid_quality(t *testing.T) {
	a := Accept("text/html;q=blah")
	mr := a.Parse()

	assert.Equal(t, 1, len(mr))
	assert.Equal(t, "text/html", mr[0].Value)
	assert.Equal(t, ParameteredMediaRangeWeight, mr[0].Weight)
}

func TestMediaRanges_should_not_remove_accept_extension(t *testing.T) {
	a := Accept("text/html;q=0.5;a=1;b=2")
	mr := a.Parse()
	assert.Equal(t, 1, len(mr))
	assert.Equal(t, "text/html;a=1;b=2", mr[0].Value)
	assert.Equal(t, 0.5, mr[0].Weight)
}

func TestMediaRanges_should_handle_precedence(t *testing.T) {
	a := Accept("text/*, text/html, text/html;level=1, */*")
	mr := a.Parse()
	assert.Equal(t, "text/html;level=1", mr[0].Value)
	assert.Equal(t, "text/html", mr[1].Value)
	assert.Equal(t, "text/*", mr[2].Value)
	assert.Equal(t, "*/*", mr[3].Value)
}

func TestMediaRanges_should_handle_precedence2(t *testing.T) {
	a := Accept("text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5")
	mr := a.Parse()

	assert.Equal(t, 5, len(mr))

	assert.Equal(t, "text/html;level=1", mr[0].Value)
	assert.Equal(t, 1.0, mr[0].Weight)

	assert.Equal(t, "text/html", mr[1].Value)
	assert.Equal(t, 0.7, mr[1].Weight)

	assert.Equal(t, "*/*", mr[2].Value)
	assert.Equal(t, 0.5, mr[2].Weight)

	assert.Equal(t, "text/html;level=2", mr[3].Value)
	assert.Equal(t, 0.4, mr[3].Weight)

	assert.Equal(t, "text/*", mr[4].Value)
	assert.Equal(t, 0.3, mr[4].Weight)
}

func TestMediaRanges_should_handle_precedence3(t *testing.T) {
	// from http://tools.ietf.org/html/rfc7231#section-5.3.2
	a := Accept("text/*, text/plain, text/plain;format=flowed, */*")
	mr := a.Parse()

	assert.Equal(t, 4, len(mr))

	assert.Equal(t, "text/plain;format=flowed", mr[0].Value)
	assert.Equal(t, 1.0, mr[0].Weight)

	assert.Equal(t, "text/plain", mr[1].Value)
	assert.Equal(t, 0.9, mr[1].Weight)

	assert.Equal(t, "text/*", mr[2].Value)
	assert.Equal(t, 0.8, mr[2].Weight)

	assert.Equal(t, "*/*", mr[3].Value)
	assert.Equal(t, 0.7, mr[3].Weight)
}

func TestMediaRanges_should_handle_precedence4(t *testing.T) {
	// from http://tools.ietf.org/html/rfc7231#section-5.3.1
	// and http://tools.ietf.org/html/rfc7231#section-5.3.2
	a := Accept("text/* ; q=0.3, text/html ; Q=0.7, text/html;level=1, text/html;level=2; q=0.4, */*; q=0.5")
	mr := a.Parse()

	assert.Equal(t, 5, len(mr))

	assert.Equal(t, "text/html;level=1", mr[0].Value)
	assert.Equal(t, 1.0, mr[0].Weight)

	assert.Equal(t, "text/html", mr[1].Value)
	assert.Equal(t, 0.7, mr[1].Weight)

	assert.Equal(t, "*/*", mr[2].Value)
	assert.Equal(t, 0.5, mr[2].Weight)

	assert.Equal(t, "text/html;level=2", mr[3].Value)
	assert.Equal(t, 0.4, mr[3].Weight)

	assert.Equal(t, "text/*", mr[4].Value)
	assert.Equal(t, 0.3, mr[4].Weight)
}
