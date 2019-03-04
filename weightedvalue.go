package negotiator

import "sort"

// WeightedValue is a value and associate weight between 0.0 and 1.0.
type WeightedValue struct {
	Value  string
	Weight float64
}

// WeightedValues is a slice of WeightedValue.
type WeightedValues []WeightedValue

// Sorted gets the values in priority order.
func (wv WeightedValues) Sorted() WeightedValues {
	sort.Sort(byWeight(wv))
	return wv
}

// Values gets the values without their weights.
func (wv WeightedValues) Values() []string {
	if wv == nil {
		return nil
	}
	ss := make([]string, len(wv))
	for i, v := range wv {
		ss[i] = v.Value
	}
	return ss
}

// ByWeight implements sort.Interface for []WeightedValue based
//on the Weight field. The data will be returned sorted decending
type byWeight []WeightedValue

func (a byWeight) Len() int           { return len(a) }
func (a byWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byWeight) Less(i, j int) bool { return a[i].Weight > a[j].Weight }
