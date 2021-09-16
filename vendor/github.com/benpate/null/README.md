# null 🚫


[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/benpate/null)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/null?style=flat-square)](https://goreportcard.com/report/github.com/benpate/null)
[![Build Status](http://img.shields.io/travis/benpate/null.svg?style=flat-square)](https://travis-ci.org/benpate/null)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/null.svg?style=flat-square)](https://codecov.io/gh/benpate/null)
![Version](https://img.shields.io/github/v/release/benpate/null?include_prereleases&style=flat-square&color=brightgreen)

## Simple library for null values in Go

This library provides simple, idiomatic primitives for nullable values in Go.  It supports Int, Bool, and Float types.

```
	// "b" is null, and ready to use
	var b null.Bool

	// Set value to false
	b.Set(false)

	// Set value to true
	b.Set(true)

	// Get the value
	b.Bool()

	// Make the value null again
	b.Unset()
```

## Pull Requests Welcome

Please use GitHub to make suggestions, pull requests, and enhancements.  We're all in this together! 🚫