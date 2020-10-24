package header

import (
	"fmt"
	"strings"
)

// MediaRange is a media range value and associated quality between 0.0 and 1.0.
// There may also be parameters (e.g. "charset") and extension values.
type MediaRange struct {
	Type, Subtype string
	Quality       float64
	Params        []KV
	Extensions    []KV
}

// mrByPrecedence implements sort.Interface for []MediaRange based
// on the precedence rules. The data will be returned sorted decending
type mrByPrecedence []MediaRange

func (a mrByPrecedence) Len() int      { return len(a) }
func (a mrByPrecedence) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a mrByPrecedence) Less(i, j int) bool {
	return a[i].StrongerThan(a[j])
}

func (mr MediaRange) StrongerThan(other MediaRange) bool {
	// qualities are floats so we don't use == directly
	if mr.Quality > other.Quality {
		return true
	} else if mr.Quality < other.Quality {
		return false
	}

	if mr.Type != "*" {
		if other.Type == "*" {
			return true
		}
		if mr.Subtype != "*" && other.Subtype == "*" {
			return true
		}
	}

	if mr.Type == other.Type {
		if mr.Subtype == other.Subtype {
			return len(mr.Params) > len(other.Params)
		}
	}
	return false
}

func (mr MediaRange) Value() string {
	return fmt.Sprintf("%s/%s", mr.Type, mr.Subtype)
}

func (mr MediaRange) String() string {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "%s/%s", mr.Type, mr.Subtype)
	for _, p := range mr.Params {
		fmt.Fprintf(buf, ";%s=%s", p.Key, p.Value)
	}
	if mr.Quality < DefaultQuality {
		fmt.Fprintf(buf, ";q=%g", mr.Quality)
	}
	for _, p := range mr.Extensions {
		fmt.Fprintf(buf, ";%s=%s", p.Key, p.Value)
	}
	return buf.String()
}
