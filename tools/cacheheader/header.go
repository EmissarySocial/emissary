// Package cacheheader provides a simple parser and serializer for http `Cache-Control` headers
package cacheheader

import (
	"net/http"
	"strconv"
	"strings"
)

// // https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
type Header struct {
	MaxAge               int64 `bson:"maxAge,omitempty"`               // indicates that the response remains fresh until N seconds after the response is generated.
	SMaxAge              int64 `bson:"sMaxAge,omitempty"`              // indicates how long the response remains fresh in a shared cache.
	NoCache              bool  `bson:"noCache,omitempty"`              // indicates that the response can be stored in caches, but the response must be validated with the origin server before each reuse.
	NoStore              bool  `bson:"noStore,omitempty"`              // indicates that any caches of any kind (private or shared) should not store this response.
	NoTransform          bool  `bson:"noTransform,omitempty"`          // indicates that any intermediary (regardless of whether it implements a cache) shouldn't transform the response contents.
	MustRevalidate       bool  `bson:"mustRevalidate,omitempty"`       // indicates that the response can be stored in caches and can be reused while fresh. If the response becomes stale, it must be validated with the origin server before reuse.
	ProxyRevalidate      bool  `bson:"proxyRevalidate,omitempty"`      // The proxy-revalidate response directive is the equivalent of must-revalidate, but specifically for shared caches only.
	MustUnderstand       bool  `bson:"mustUnderstand,omitempty"`       // indicates that a cache should store the response only if it understands the requirements for caching based on status code.
	Private              bool  `bson:"private,omitempty"`              // indicates that the response can be stored only in a private cache (e.g. local caches in browsers).
	Public               bool  `bson:"public,omitempty"`               // indicates that the response can be stored in a shared cache.
	Immutable            bool  `bson:"immutable,omitempty"`            // indicates that the response will not be updated while it's fresh.
	StaleWhileRevalidate int64 `bson:"staleWhileRevalidate,omitempty"` // indicates that the cache could reuse a stale response while it revalidates it to a cache.
	StaleIfError         int64 `bson:"staleIfError,omitempty"`         // indicates that the cache can reuse a stale response when an upstream server generates an error

	asPublicCache bool // Treats this header as if being parsed by a public cache.
}

// Parse generates a new Directive structure from an http.Header
func Parse(header http.Header, options ...HeaderOption) Header {
	combinedValue := strings.Join(header[HeaderCacheControl], ", ")
	return ParseString(combinedValue, options...)
}

// ParseString generates a new Directive structure from a Cache-Control string
func ParseString(value string, options ...HeaderOption) Header {

	result := Header{}

	for _, option := range options {
		option(&result)
	}

	value = strings.ToLower(value)
	items := strings.Split(value, ",")

	for _, item := range items {

		item = strings.TrimSpace(item)
		switch directive, argument, _ := strings.Cut(item, "="); directive {

		case DirectiveMaxAge:
			if maxAge, err := strconv.ParseInt(argument, 10, 64); err == nil {
				result.MaxAge = maxAge
			}
		case DirectiveSMaxAge:
			if sMaxAge, err := strconv.ParseInt(argument, 10, 64); err == nil {
				result.SMaxAge = sMaxAge
			}

		case DirectiveNoCache:
			result.NoCache = true

		case DirectiveNoStore:
			result.NoStore = true

		case DirectiveNoTransform:
			result.NoTransform = true

		case DirectiveMustRevalidate:
			result.MustRevalidate = true

		case DirectiveProxyRevalidate:
			result.ProxyRevalidate = true

		case DirectiveMustUnderstand:
			result.MustUnderstand = true

		case DirectivePrivate:
			result.Private = true

		case DirectivePublic:
			result.Public = true

		case DirectiveImmutable:
			result.Immutable = true

		case DirectiveStaleWhileRevalidate:
			if staleWhileRevalidate, err := strconv.ParseInt(argument, 10, 64); err == nil {
				result.StaleWhileRevalidate = staleWhileRevalidate
			}

		case DirectiveStaleIfError:
			if staleIfError, err := strconv.ParseInt(argument, 10, 64); err == nil {
				result.StaleIfError = staleIfError
			}
		}
	}

	return result
}

// String returns the string representation of this directive.
func (header Header) String() string {

	directive := make([]string, 0)

	if header.MaxAge > 0 {
		directive = append(directive, DirectiveMaxAge+"="+strconv.FormatInt(header.MaxAge, 10))
	}

	if header.SMaxAge > 0 {
		directive = append(directive, DirectiveSMaxAge+"="+strconv.FormatInt(header.SMaxAge, 10))
	}

	if header.NoCache {
		directive = append(directive, DirectiveNoCache)
	}

	if header.NoStore {
		directive = append(directive, DirectiveNoStore)
	}

	if header.NoTransform {
		directive = append(directive, DirectiveNoTransform)
	}

	if header.MustRevalidate {
		directive = append(directive, DirectiveMustRevalidate)
	}

	if header.ProxyRevalidate {
		directive = append(directive, DirectiveProxyRevalidate)
	}

	if header.MustUnderstand {
		directive = append(directive, DirectiveMustUnderstand)
	}

	if header.Private {
		directive = append(directive, DirectivePrivate)
	}

	if header.Public {
		directive = append(directive, DirectivePublic)
	}

	if header.Immutable {
		directive = append(directive, DirectiveImmutable)
	}

	if header.StaleWhileRevalidate > 0 {
		directive = append(directive, DirectiveStaleWhileRevalidate+"="+strconv.FormatInt(header.StaleWhileRevalidate, 10))
	}

	if header.StaleIfError > 0 {
		directive = append(directive, DirectiveStaleIfError+"="+strconv.FormatInt(header.StaleIfError, 10))
	}

	return strings.Join(directive, ", ")
}

// IsNil returns TRUE if no values are set within this header value.
func (header Header) IsNil() bool {

	if header.MaxAge > 0 {
		return false
	}

	if header.SMaxAge > 0 {
		return false
	}

	if header.NoCache {
		return false
	}

	if header.NoStore {
		return false
	}

	if header.NoTransform {
		return false
	}

	if header.MustRevalidate {
		return false
	}

	if header.ProxyRevalidate {
		return false
	}

	if header.MustUnderstand {
		return false
	}

	if header.Private {
		return false
	}

	if header.Public {
		return false
	}

	if header.Immutable {
		return false
	}

	if header.StaleWhileRevalidate > 0 {
		return false
	}

	if header.StaleIfError > 0 {
		return false
	}

	return true
}

// NotNil returns TRUE if at least one value is set in the header
func (header Header) NotNil() bool {
	return !header.IsNil()
}

// IsCacheAllowed returns TRUE if this header's settings allows a value to be cached
func (header Header) IsCacheAllowed() bool {

	if header.NoCache {
		return false
	}

	if header.NoStore {
		return false
	}

	if header.MaxAge == 0 {
		return false
	}

	if header.asPublicCache && header.Private {
		return false
	}

	return true
}

// NotCacheAllowed returns TRUE if this header's settings DO NOT ALLOW a value to be cached
func (header Header) NotCacheAllowed() bool {
	return !header.IsCacheAllowed()
}
