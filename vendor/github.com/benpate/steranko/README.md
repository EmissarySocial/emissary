# Steranko 🔐

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/benpate/steranko)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/steranko?style=flat-square)](https://goreportcard.com/report/github.com/benpate/steranko)
[![Build Status](http://img.shields.io/travis/benpate/steranko.svg?style=flat-square)](https://travis-ci.com/benpate/steranko)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/steranko.svg?style=flat-square)](https://codecov.io/gh/benpate/steranko)

## Website Authentication/Authorization for Go

**This project is a work-in-progress, and should NOT be used by ANYONE, for ANY PURPOSE, under ANY CIRCUMSTANCES.  It is GUARANTEED to blow up your computer, send your cat into an infinite loop, and combine your hot and cold laundry into a single cycle.**

Steranko is an embeddable library that manages user authentication, and authorization.  You can configure it at run time (or compile time) to meet your specific project needs.

To use Steranko, you have to implement two tiny interfaces in your code, then wire Steranko's handlers into your HTTP server.

```go
s := steranko.New(userService, steranko.Conig{
  Tokens: "cookie:auth",
  PasswordSchema: `{"type":"string", "minLength":20}`
})

s.Register(echo)
```


## Project Goals

* Create a configurable, open source authentication/authorization system in Go.
* Hashed passwords using bcrypt
* Automatically upgrade password encryption cost on signin.
* Lock out user accounts after N failed attempts.
* Maintain security with [JWT tokens](https://jwt.io/)

* Password strength checking (via JSON-Schema extensions)
* Password vulnerability via HaveIBeenPwned API.

### Possible future additions
* Identify malicious users with a (relatively) invisible CAPTCHA system
  * Track javascript events during signup (keyup, keydown, mousemove, click)
  * Track timing of events.  They must not be too fast, or too consistent.
  * Something to prevent requests from being forwarded to an actual human.
  * Math problems?
  * Geolocation.

## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making Steranko better, send in a pull request.  We're all in this together! 🔐