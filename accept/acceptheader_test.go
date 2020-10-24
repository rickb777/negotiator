package accept

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseAcceptHeader_parses_single(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("application/json")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Type).To(Equal("application"))
	g.Expect(mr[0].Subtype).To(Equal("json"))
	g.Expect(mr[0].Quality).To(Equal(DefaultQuality))
}

func TestParseAcceptHeader_preserves_case_of_mediaRange(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("application/CEA")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Type).To(Equal("application"))
	g.Expect(mr[0].Subtype).To(Equal("CEA"))
}

func TestParseAcceptHeader_defaults_quality_if_not_explicit(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("text/plain")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Quality).To(Equal(DefaultQuality))
}

func TestParseAcceptHeader_should_parse_quality(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("application/json;q=0.9")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Type).To(Equal("application"))
	g.Expect(mr[0].Subtype).To(Equal("json"))
	g.Expect(mr[0].Quality).To(Equal(0.9))
}

func TestParseAcceptHeader_sorts_by_decending_quality(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("application/json;q=0.8, application/xml, application/*;q=0.1")

	g.Expect(len(mr)).To(Equal(3))

	g.Expect(mr[0].Type).To(Equal("application"))
	g.Expect(mr[0].Subtype).To(Equal("xml"))
	g.Expect(mr[0].Quality).To(Equal(DefaultQuality))

	g.Expect(mr[1].Type).To(Equal("application"))
	g.Expect(mr[1].Subtype).To(Equal("json"))
	g.Expect(mr[1].Quality).To(Equal(0.8))

	g.Expect(mr[2].Type).To(Equal("application"))
	g.Expect(mr[2].Subtype).To(Equal("*"))
	g.Expect(mr[2].Quality).To(Equal(0.1))
}

func TestMediaRanges_should_ignore_invalid_quality(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("text/html;q=blah")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Type).To(Equal("text"))
	g.Expect(mr[0].Subtype).To(Equal("html"))
	g.Expect(mr[0].Quality).To(Equal(DefaultQuality))
	g.Expect(mr[0].Params).To(HaveLen(0))
}

func TestMediaRanges_should_not_remove_accept_extension(t *testing.T) {
	g := NewGomegaWithT(t)
	mr := ParseAcceptHeader("text/html;q=0.5;a=1;b=2")

	g.Expect(len(mr)).To(Equal(1))
	g.Expect(mr[0].Type).To(Equal("text"))
	g.Expect(mr[0].Subtype).To(Equal("html"))
	g.Expect(mr[0].Quality).To(Equal(0.5))
	g.Expect(mr[0].Params).To(ConsistOf(KV{"a", "1"}, KV{"b", "2"}))
}

// If more than one media range applies to a
// given type, the most specific reference has precedence
func TestMediaRanges_should_handle_precedence(t *testing.T) {
	g := NewGomegaWithT(t)
	// from https://tools.ietf.org/html/rfc7231#section-5.3.2
	c := "text/*, text/plain, text/plain;format=flowed, */*"
	mr := ParseAcceptHeader(c)

	g.Expect(len(mr)).To(Equal(4))
	g.Expect(mr[0]).To(Equal(MediaRange{
		Type:    "text",
		Subtype: "plain",
		Quality: DefaultQuality,
		Params:  []KV{{"format", "flowed"}},
	}), c)
	g.Expect(mr[1]).To(Equal(MediaRange{
		Type:    "text",
		Subtype: "plain",
		Quality: DefaultQuality,
	}), c)
	g.Expect(mr[2]).To(Equal(MediaRange{
		Type:    "text",
		Subtype: "*",
		Quality: DefaultQuality,
	}), c)
	g.Expect(mr[3]).To(Equal(MediaRange{
		Type:    "*",
		Subtype: "*",
		Quality: DefaultQuality,
	}), c)
}

func TestMediaRanges_should_handle_quality_precedence(t *testing.T) {
	g := NewGomegaWithT(t)
	cases := []string{
		// each example has a distinct quality for each part
		"text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5",
		"text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5, text/*;q=0.3",
		"text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5, text/*;q=0.3, text/html;q=0.7",
		"text/html;level=2;q=0.4, */*;q=0.5, text/*;q=0.3, text/html;q=0.7, text/html;level=1",
	}
	for _, c := range cases {
		mr := ParseAcceptHeader(c)
		g.Expect(5, len(mr))

		g.Expect(mr[0]).To(Equal(MediaRange{
			Type:    "text",
			Subtype: "html",
			Quality: DefaultQuality,
			Params:  []KV{{"level", "1"}},
		}), c)

		g.Expect(mr[1]).To(Equal(MediaRange{
			Type:    "text",
			Subtype: "html",
			Quality: 0.7,
		}), c)

		g.Expect(mr[2]).To(Equal(MediaRange{
			Type:    "*",
			Subtype: "*",
			Quality: 0.5,
		}), c)

		g.Expect(mr[3]).To(Equal(MediaRange{
			Type:    "text",
			Subtype: "html",
			Quality: 0.4,
			Params:  []KV{{"level", "2"}},
		}), c)

		g.Expect(mr[4]).To(Equal(MediaRange{
			Type:    "text",
			Subtype: "*",
			Quality: 0.3,
		}), c)
	}
}

//func TestMediaRanges_should_handle_qualities(t *testing.T) {
//	g := NewGomegaWithT(t)
//	// from http://tools.ietf.org/html/rfc7231#section-5.3.2
//	// and errata https://www.rfc-editor.org/errata/rfc7231
//
//	cases := []string{
//		"text/*;q=0.3, text/plain;q=0.7, text/plain;format=flowed, text/plain;format=fixed;q=0.4, */*;q=0.5",
//		//"text/plain;q=0.7, text/plain;format=flowed, text/plain;format=fixed;q=0.4, */*;q=0.5, text/*;q=0.3",
//		//"text/plain;format=flowed, text/plain;format=fixed;q=0.4, */*;q=0.5, text/*;q=0.3, text/plain;q=0.7",
//		//"text/plain;format=fixed;q=0.4, */*;q=0.5, text/*;q=0.3, text/plain;q=0.7, text/plain;format=flowed",
//	}
//	for _, c := range cases {
//		mr := ParseAcceptHeader(c)
//		g.Expect(5, len(mr))
//
//		g.Expect("text/plain;format=flowed", mr[0].Value, c)
//		g.Expect(1.0, mr[0].Quality, c)
//
//		g.Expect("text/plain", mr[1].Value, c)
//		g.Expect(0.7, mr[1].Quality, c)
//
//		g.Expect("*/*", mr[3].Value, c)
//		g.Expect(0.5, mr[3].Quality, c)
//
//		g.Expect("text/plain;format=fixed", mr[3].Value, c)
//		g.Expect(0.4, mr[3].Quality, c)
//
//		g.Expect("text/html;level=2", mr[3].Value, c)
//		g.Expect(0.4, mr[3].Quality, c)
//
//		g.Expect("text/html", mr[1].Value, c)
//		g.Expect(0.3, mr[1].Quality, c)
//	}
//}

//func TestMediaRanges_should_ignore_case_of_quality_and_whitespace(t *testing.T) {
//	g := NewGomegaWithT(t)
//	// from http://tools.ietf.org/html/rfc7231#section-5.3.1
//	// and http://tools.ietf.org/html/rfc7231#section-5.3.2
//	mr := ParseAcceptHeader("text/* ; q=0.3, text/html ; Q=0.7, text/html;level=2; q=0.4, */*; q=0.5")
//
//	g.Expect(4, len(mr))
//
//	g.Expect("text/html", mr[0].Value)
//	g.Expect(0.7, mr[0].Quality)
//
//	g.Expect("*/*", mr[1].Value)
//	g.Expect(0.5, mr[1].Quality)
//
//	g.Expect("text/html;level=2", mr[2].Value)
//	g.Expect(0.4, mr[2].Quality)
//
//	g.Expect("text/*", mr[3].Value)
//	g.Expect(0.3, mr[3].Quality)
//}
