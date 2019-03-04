package negotiator

import (
	"strconv"
	"strings"
)

const (
	// ParameteredMediaRangeWeight is the default weight of a media range with an
	// Accept-param
	ParameteredMediaRangeWeight float64 = 1.0 //e.g text/html;level=1
	// TypeSubtypeMediaRangeWeight is the default weight of a media range with
	// type and subtype defined
	TypeSubtypeMediaRangeWeight float64 = 0.9 //e.g text/html
	// TypeStarMediaRangeWeight is the default weight of a media range with a type
	// defined but * for subtype
	TypeStarMediaRangeWeight float64 = 0.8 //e.g text/*
	// StarStarMediaRangeWeight is the default weight of a media range with any
	// type or any subtype defined
	StarStarMediaRangeWeight float64 = 0.7 //e.g */*
)

// Accept is an http accept header value, e.g. as in "Accept-Language: en-GB,en;q=0.5".
type Accept string

// Parse returns a (multi-valued) header in priority order. A blank input returns nil.
func (accept Accept) Parse() WeightedValues {
	if accept == "" {
		return nil
	}

	var values WeightedValues
	mrs := strings.Split(string(accept), ",")

	for _, mr := range mrs {
		mrAndAcceptParam := strings.Split(mr, ";")
		//if no Accept-param
		if len(mrAndAcceptParam) == 1 {
			values = append(values, handleMediaRangeNoAcceptParams(mrAndAcceptParam[0]))
			continue
		}

		values = append(values, handleMediaRangeWithAcceptParams(mrAndAcceptParam[0], mrAndAcceptParam[1:]))
	}

	//If no Accept header field is present, then it is assumed that the client
	//accepts all media types. If an Accept header field is present, and if the
	//server cannot send a response which is acceptable according to the combined
	//Accept field value, then the server SHOULD send a 406 (not acceptable)
	//response.

	return values.Sorted()
}

func handleMediaRangeWithAcceptParams(mediaRange string, acceptParams []string) WeightedValue {
	wv := WeightedValue{
		Value:  strings.TrimSpace(mediaRange),
		Weight: ParameteredMediaRangeWeight,
	}

	for index := 0; index < len(acceptParams); index++ {
		ap := strings.ToLower(acceptParams[index])
		if isQualityAcceptParam(ap) {
			wv.Weight = parseQuality(ap)
		} else {
			wv.Value = strings.Join([]string{wv.Value, acceptParams[index]}, ";")
		}
	}
	return wv
}

func isQualityAcceptParam(acceptParam string) bool {
	return strings.Contains(acceptParam, "q=")
}

func parseQuality(acceptParam string) float64 {
	weight, err := strconv.ParseFloat(strings.SplitAfter(acceptParam, "q=")[1], 64)
	if err != nil {
		weight = 1.0
	}
	return weight
}

func handleMediaRangeNoAcceptParams(mediaRange string) WeightedValue {
	wv := WeightedValue{
		Value:  strings.TrimSpace(mediaRange),
		Weight: 1.0,
	}

	typeSubtype := strings.Split(wv.Value, "/")
	if len(typeSubtype) == 2 {
		switch {
		//a type of * with a non-star subtype is invalid, so if the type is
		//star the assume that the subtype is too
		case typeSubtype[0] == "*": //&& typeSubtype[1] == "*":
			wv.Weight = StarStarMediaRangeWeight
			break
		case typeSubtype[1] == "*":
			wv.Weight = TypeStarMediaRangeWeight
			break
		case typeSubtype[1] != "*":
			wv.Weight = TypeSubtypeMediaRangeWeight
			break
		}
	} //else the weight remains 1.0

	return wv
}
