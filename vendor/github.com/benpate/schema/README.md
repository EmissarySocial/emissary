# schema 👍


[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/benpate/schema)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/schema?style=flat-square)](https://goreportcard.com/report/github.com/benpate/schema)
[![Build Status](http://img.shields.io/travis/benpate/schema.svg?style=flat-square)](https://travis-ci.com/benpate/schema)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/schema.svg?style=flat-square)](https://codecov.io/gh/benpate/schema)
![Version](https://img.shields.io/github/v/release/benpate/schema?include_prereleases&style=flat-square&color=brightgreen)

## Simplified JSON-Schema

This is my first attempt to create a simplified, minimal implementation of JSON-Schema that is fast and has an easy API.  If you're looking for a complete, rigorous implementation of JSON-Schema, you should try another tool.

## What it Does

This library implements a sub-set of the [JSON-Schema specification](http://json-schema.org)

### What's Included

* Unmarshal schema from JSON
* Array, Boolean, Integer, Number, Object, and String type validators.
* Custom Format rules
* Happy API for accessing schema information, and walking a schema tree with a [JSON-Pointer](https://tools.ietf.org/html/rfc6901)

### What's Left Out

* References
* Loading remote Schemas by URI.


## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making Rosetta better, send in a pull request.  We're all in this together! 👍