// Package headers writes the response headers that HEAD and GET must agree on.
//
// A single Emissary URL can serve two documents -- an HTML page for a person, or an ActivityStreams
// document for a federated peer -- and RFC 9110 section 9.3.2 requires HEAD to report the same
// header fields the equivalent GET would send. Both verbs derive those fields here, so they cannot
// drift apart.
//
// Which document a request asks for is decided by hannibal, which owns the ActivityPub reading of
// the "Accept" header for every consumer. This package turns that answer into a Variant, and a
// Variant into headers.
package headers

import (
	"net/http"
	"strings"
	"time"

	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
)

// Variant is one of the document formats Emissary serves from a single URL.
type Variant int

const (
	// VariantHTML is the default representation -- a web page for a person.
	VariantHTML Variant = iota

	// VariantActivityPub is the ActivityStreams 2.0 / JSON-LD representation, for federated peers.
	VariantActivityPub
)

// Validator is the object surface required to build cache validators. Anything embedding
// journal.Journal satisfies it, including model.Stream, model.User, and data.Object.
type Validator interface {

	// ETag returns an opaque revision marker that changes whenever the object changes
	ETag() string

	// Updated returns the time of the last change, in Unix milliseconds
	Updated() int64
}

// ContentTypeHTML is the Content-Type an HTML response carries.
const ContentTypeHTML = "text/html; charset=UTF-8" // Exactly what Echo's ctx.HTML writes, so HEAD can match GET without rendering

// VaryHTML is the Vary header an HTML response carries.
const VaryHTML = "Cookie, HX-Request, Accept" // The page differs per session, per htmx partial request, and per variant

// VaryActivityPub is the Vary header an ActivityStreams response carries.
const VaryActivityPub = "Accept"

// variantTokens name each Variant inside an entity-tag, so that two documents served from one URL
// can never share a validator.
var variantTokens = map[Variant]string{
	VariantHTML:        "html",
	VariantActivityPub: "as2",
}

// VariantOf returns the Variant that best matches the request's "Accept" header.
func VariantOf(request *http.Request) Variant {

	// HTML is the default, so it is also where an absent, empty, or unrecognized header lands.
	if hannibal.IsActivityPubRequest(request) {
		return VariantActivityPub
	}

	return VariantHTML
}

// ContentType returns the Content-Type a response of this Variant must carry.
func ContentType(variant Variant) string {

	// Fixed rather than echoed back from the request: preference ordering can select the `ld+json`
	// entry sitting beside Mastodon's `activity+json`, and swapping which one goes on the wire gains
	// nothing.
	if variant == VariantActivityPub {
		return vocab.ContentTypeActivityPub
	}

	return ContentTypeHTML
}

// Vary returns the Vary header a response of this Variant must carry.
func Vary(variant Variant) string {

	if variant == VariantActivityPub {
		return VaryActivityPub
	}

	return VaryHTML
}

// ETag returns the entity-tag identifying one Variant of one object.
func ETag(variant Variant, object Validator) string {

	// RULE: the Variant is part of the tag. An entity-tag identifies a specific representation
	// (RFC 9110 section 8.8.1), so the HTML and ActivityStreams documents at one URL must not share
	// one -- a client holding the HTML tag would otherwise be answered 304 when it asks for AS2.
	return EntityTag(object.ETag(), variantTokens[variant])
}

// LastModified returns the object's modification time, formatted for an HTTP header.
func LastModified(object Validator) string {

	// Normalizing to UTC matters because http.TimeFormat ends in a literal "GMT" regardless of the
	// value's location -- a local time would be labelled GMT and read as hours out.
	return time.UnixMilli(object.Updated()).UTC().Format(http.TimeFormat)
}

// EntityTag assembles the provided parts into a weak entity-tag.
func EntityTag(parts ...string) string {

	// RULE: the tag is WEAK. It is built from a database revision, which does not change when a
	// domain edits a template or a deployment changes a serializer -- so these tags promise semantic
	// equivalence, not byte-equality. Making them strong requires folding in a build identifier;
	// see projects/DEPLOY-IDENTIFIER.md.
	for index, part := range parts {
		parts[index] = sanitize(part)
	}

	return `W/"` + strings.Join(parts, "-") + `"`
}

// sanitize strips the characters that may not appear inside an entity-tag
// (RFC 9110 section 8.8.3: etagc excludes DQUOTE, and control characters are not permitted).
func sanitize(value string) string {

	result := make([]byte, 0, len(value))

	for index := 0; index < len(value); index++ {
		if character := value[index]; (character > 0x20) && (character != '"') && (character != 0x7F) {
			result = append(result, character)
		}
	}

	return string(result)
}

// SetVariant writes the headers that describe which document is being served.
func SetVariant(header http.Header, variant Variant) {
	header.Set(vocab.ContentType, ContentType(variant))
	header.Set("Vary", Vary(variant))
}

// SetValidators writes the headers a client needs in order to revalidate this document later.
func SetValidators(header http.Header, variant Variant, object Validator) {
	header.Set("ETag", ETag(variant, object))
	header.Set("Last-Modified", LastModified(object))
}

// SetAll writes every header that HEAD and GET must agree on.
func SetAll(header http.Header, variant Variant, object Validator) {
	SetVariant(header, variant)
	SetValidators(header, variant, object)
}
