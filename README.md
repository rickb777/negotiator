# Negotiator 

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg)](http://pkg.go.dev/github.com/rickb777/negotiator)
[![Build Status](https://travis-ci.org/rickb777/negotiator.svg?branch=master)](https://travis-ci.org/rickb777/negotiator/builds)
[![Issues](https://img.shields.io/github/issues/rickb777/negotiator.svg)](https://github.com/rickb777/negotiator/issues)

This is a libary that handles content negotiation in web applications written in Go.

## Usage

### Simple
To return JSON/XML out of the box simple put this in your route handler:
```
func getUser(w http.ResponseWriter, req *http.Request) {
    user := &User{"Joe","Bloggs"}
    if err := negotiator.Negotiate(w, req, user); err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

### Custom

To add your own negotiator, for example you want to write a PDF with your model, do the following:

1) Create a type that conforms to the [ResponseProcessor](https://github.com/rickb777/negotiator/blob/master/responseprocessor.go) interface

2) Call `negotiator.New(responseProcessors ...ResponseProcessor)` and pass in a your custom processor. When your request handler calls `negotiator.Negotiate(w,req,model,errorHandler)` it will render a PDF if your Accept header defined it wanted a PDF response.

### When a request is Not Acceptable

You will create a `Negotiator` with one or more response processors. If a request is handled that is not claimed by and processor, a Not Acceptable (406) response is returned. This uses the default `http.Error` function to render the response, but a custom error handler can be plugged in using `Negotiator.With(myHandler)`.

## Accept Handling

The `Accept` type can be used for other 'Accept' headers too (e.g. `Accept-Language`)'. Simply type-convert a header string to `Accept` and call its `Parse()` method, e.g.

```
    // handle Accept-Language
    accept := negotiator.Accept("en-GB,en;q=0.5")
    languages := accept.Parse().Values()
    // this will contain {"en-GB", "en"}
```

This can be used for `Accept-Language`, `Accept-Charset` and `Accept-Encoding`, as well as `Accept`. The negotiator in this API uses the `Accept` header only, though.

## Acknowledgement

Many thanks to Jonathan Channon (https://github.com/jchannon) for the original concepts and work on which this was based.
  