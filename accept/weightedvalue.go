package accept

import "fmt"

// KV holds a parameter with a key and optional value.
type KV struct {
	Key, Value string
}

// PrecedenceValue is a value and associate quality between 0.0 and 1.0
type PrecedenceValue struct {
	Value      string
	Quality    float64
	Params     []KV
	Extensions []KV
}

// wvByPrecedence implements sort.Interface for []PrecedenceValue based
// on the precedence rules. The data will be returned sorted decending
type wvByPrecedence []PrecedenceValue

func (a wvByPrecedence) Len() int      { return len(a) }
func (a wvByPrecedence) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a wvByPrecedence) Less(i, j int) bool {
	// qualities are floats so we don't use == directly
	if a[i].Quality > a[j].Quality {
		return true
	} else if a[i].Quality < a[j].Quality {
		return false
	}
	if a[i].Value == a[j].Value {
		return len(a[i].Params) > len(a[j].Params)
	}
	return false
}

//-------------------------------------------------------------------------------------------------

// MediaRange is a media range value and associate weight between 0.0 and 1.0
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
	// qualities are floats so we don't use == directly
	if a[i].Quality > a[j].Quality {
		return true
	} else if a[i].Quality < a[j].Quality {
		return false
	}

	if a[i].Type != "*" {
		if a[j].Type == "*" {
			return true
		}
		if a[i].Subtype != "*" && a[j].Subtype == "*" {
			return true
		}
	}

	if a[i].Type == a[j].Type {
		if a[i].Subtype == a[j].Subtype {
			return len(a[i].Params) > len(a[j].Params)
		}
	}
	return false
}

func (mr MediaRange) Value() string {
	return fmt.Sprintf("%s/%s", mr.Type, mr.Subtype)
}
