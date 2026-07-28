package model

import (
	"net/url"
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/uri"
	"golang.org/x/net/idna"
)

// maxDomainKeys caps how many hostname suffixes a single document contributes.
// A real hostname has a handful of labels; the cap only bounds pathological inputs.
const maxDomainKeys = 10

// RuleMatchKey returns the canonical match key for a Rule of the given Type and Trigger.
//
// A Rule and every document it matches produce the SAME key, so matching becomes an indexed
// equality lookup instead of a scan. The key is "<TYPE>:<normalized trigger>"; the fixed literal
// prefix (the Type constant) makes cross-type collisions impossible. An unrecognized Type, or a
// Trigger that normalizes to nothing, yields an empty key -- which matches nothing.
//
// Per-type normalizers are NOT interchangeable: `ToToken` would shred an actor URI, and an actor
// URI is not a hostname. Whatever runs here MUST also run on the document side (DocumentMatchKeys),
// or a Rule silently stops matching.
func RuleMatchKey(ruleType string, trigger string) string {

	switch ruleType {

	case RuleTypeActor:
		if actor := normalizeActorURI(trigger); actor != "" {
			return RuleTypeActor + ":" + actor
		}

	case RuleTypeDomain:
		if host := normalizeHostname(trigger); host != "" {
			return RuleTypeDomain + ":" + host
		}

	case RuleTypeTag:
		if tag := ToToken(trigger); tag != "" {
			return RuleTypeTag + ":" + tag
		}
	}

	// Unknown type or empty trigger federates as nothing.
	return ""
}

// DocumentMatchKeys returns every match key that the provided document's own identity can produce:
// its responsible identities -- `actor` (Activities) and `attributedTo` (Objects) -- each as an ACTOR
// key plus the DOMAIN keys of its host, and each of its Hashtag tags (as TAG keys). A Rule matches
// this document when its MatchKey is in this set, so the engine never re-scans -- it intersects two
// key sets.
//
// This reads only fetch-free properties. `ActorID()` and `AttributedTo().ID()` short-circuit on
// bare-string values, and tag names are read only from object-valued tags (a bare-string tag is
// skipped, because reading its name would fetch it over the network).
func DocumentMatchKeys(document streams.Document) []string {

	result := make([]string, 0)

	// Keys derived from the document's responsible identities. Objects name their author with
	// `attributedTo`, Activities with `actor` -- a document may carry either or both, so both are
	// keyed (skipping the duplicate when a self-attributed document repeats the same identity).
	actorID := document.ActorID()
	result = append(result, ActorMatchKeys(actorID)...)

	if authorID := document.AttributedTo().ID(); authorID != actorID {
		result = append(result, ActorMatchKeys(authorID)...)
	}

	// A TAG key for each Hashtag on the document. Mentions and Emoji are deliberately excluded
	// (D12) so a TAG rule for "alice" cannot match a post that merely mentions @alice.
	for tag := document.Tag(); tag.NotNil(); tag = tag.Next() {

		if tag.IsString() {
			continue
		}

		if tag.Type() != vocab.LinkTypeHashtag {
			continue
		}

		if token := ToToken(tag.Name()); token != "" {
			result = append(result, RuleTypeTag+":"+token)
		}
	}

	return result
}

// ActorMatchKeys returns the ACTOR and DOMAIN keys an actor URI can match: the actor itself, plus
// every host suffix. This is the wire gate's key set -- the inbox gate blocks by WHO is talking
// (ACTOR/DOMAIN), never by content (TAG), so it deliberately excludes the tag keys that
// DocumentMatchKeys adds. An empty actorID contributes nothing.
func ActorMatchKeys(actorID string) []string {

	if actorID == "" {
		return make([]string, 0)
	}

	result := make([]string, 0)

	if actor := normalizeActorURI(actorID); actor != "" {
		result = append(result, RuleTypeActor+":"+actor)
	}

	return append(result, domainMatchKeys(actorID)...)
}

// DomainMatchKeys returns the DOMAIN keys for a value's host: one per dot-boundary suffix. Used by
// the wire gate to extend a DOMAIN block to the delivering server named in a signature's keyId,
// which is a bare key URL, not an actor.
func DomainMatchKeys(value string) []string {
	return domainMatchKeys(value)
}

// normalizeActorURI lower-cases the scheme and host of an actor URI while preserving the path, so
// that a host-case variation cannot alias past an ACTOR rule. A value that does not parse as a URL
// is returned trimmed but otherwise unchanged.
func normalizeActorURI(value string) string {

	value = strings.TrimSpace(value)

	parsed, err := url.Parse(value)

	if (err != nil) || (parsed.Host == "") {
		return value
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	return parsed.String()
}

// normalizeHostname reduces any URL-ish value to its bare hostname, lower-cased and converted to
// ASCII/Punycode -- so a Unicode DOMAIN rule ("café.com") produces the same key as the wire host
// ("xn--caf-dma.com"). NOTE: uri.NormalizeHost does NOT punycode; the deep-fix home for this is
// benpate/uri, but it lives here until that lands.
func normalizeHostname(value string) string {

	host := uri.NormalizeHost(value)

	if host == "" {
		return ""
	}

	if ascii, err := idna.ToASCII(host); err == nil {
		return ascii
	}

	return host
}

// domainMatchKeys returns a DOMAIN key for every dot-boundary suffix of the value's hostname, so a
// rule on "evil.com" matches an actor at "sub.evil.com" (suffix present) but not "notevil.com"
// (suffix absent -- the dot boundary is what the old bare HasSuffix lacked).
func domainMatchKeys(value string) []string {

	host := normalizeHostname(value)

	if host == "" {
		return nil
	}

	labels := strings.Split(host, ".")

	// Bound pathological inputs by keeping only the innermost labels (a rule targets a short,
	// registrable suffix, which is always near the tail).
	if len(labels) > maxDomainKeys {
		labels = labels[len(labels)-maxDomainKeys:]
	}

	result := make([]string, 0, len(labels))

	for index := range labels {
		result = append(result, RuleTypeDomain+":"+strings.Join(labels[index:], "."))
	}

	return result
}
