# Negotiator 

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg)](http://pkg.go.dev/github.com/rickb777/negotiator)
[![Build Status](https://travis-ci.org/rickb777/negotiator.svg?branch=master)](https://travis-ci.org/rickb777/negotiator/builds)
[![Issues](https://img.shields.io/github/issues/rickb777/negotiator.svg)](https://github.com/rickb777/negotiator/issues)

This is a library that handles content negotiation in HTTP server applications written in Go.

## Usage

### Simple

To return JSON/XML out of the box simple put this in your route handler:
```
import "github.com/rickb777/negotiator"
...
func getUser(w http.ResponseWriter, req *http.Request) {
    user := &User{"Joe","Bloggs"}
    n := negotiator.NewWithJSONAndXML()
    n.MustNegotiate(w, req, negotiator.Offer{Data: user})
}
```

### Custom

To add your own negotiator, for example you want to write a PDF with your model, do the following:

1) Create a type that conforms to the [ResponseProcessor](https://github.com/rickb777/negotiator/blob/master/responseprocessor.go) interface

2) Where you call `negotiator.New(responseProcessors ...ResponseProcessor)`, pass in a your custom processor. When your request handler calls `negotiator.Negotiate(w, req, offers...)` it will render a PDF if your Accept header defined it wanted a PDF response.

### When a request is Not Acceptable

Having created a `Negotiator` with one or more response processors, if a request is handled that is not claimed by and processor, a Not Acceptable (406) response is returned. 

By default, this uses the standard `http.Error` function (from `net/http`) to render the response, If needed, a custom error handler can be plugged in using `Negotiator.WithErrorHandler(myHandler)`.

## Accept Handling

The `Accept` header is parsed using `header.ParseMediaRanges()`, which returns the slice of media ranges, e.g.

```
    // handle Accept-Language
    mediaRanges := header.ParseMediaRanges("application/json;q=0.8, application/xml, application/*;q=0.1")
```

The resulting slice is sorted according to precedence and quality rules, so in this example the order is {"application/xml", "application/json", "application/*"} because the middle item has an implied quality of 1, whereas the first item has a lower quality.

The other content-negotiation headers, `Accept-Language`, `Accept-Charset`, `Accept-Encoding`, are handled by the `header.Parse` method, e.g.

```
    // handle Accept-Language
    acceptLanguages := header.Parse("en-GB,en;q=0.5")
```

This will contain {"en-GB", "en"} in a `header.PrecedenceValues` slice, sorted according to precedence rules.

This can be used for `Accept-Language`, `Accept-Charset` and `Accept-Encoding`, as well as `Accept`. The negotiator in this API uses the `Accept` header only, though.

## Acknowledgement

Many thanks to Jonathan Channon (https://github.com/jchannon) for the original concepts and work on which this was based.
  