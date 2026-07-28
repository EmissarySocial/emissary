package model

import (
	"strings"
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRuleMatchKey_Actor pins ACTOR normalization: the host is lower-cased (so case cannot alias
// past a block) while the path is preserved (some servers route case-sensitive paths).
func TestRuleMatchKey_Actor(t *testing.T) {
	require.Equal(t, "ACTOR:https://example.com/@bob", RuleMatchKey(RuleTypeActor, "https://example.com/@bob"))
	require.Equal(t, "ACTOR:https://example.com/@Bob", RuleMatchKey(RuleTypeActor, "https://Example.COM/@Bob"))
}

// TestRuleMatchKey_Domain pins DOMAIN normalization: lower-cased, port/scheme stripped, and
// Unicode converted to Punycode so an IDN rule matches the ASCII wire host.
func TestRuleMatchKey_Domain(t *testing.T) {
	require.Equal(t, "DOMAIN:evil.com", RuleMatchKey(RuleTypeDomain, "evil.com"))
	require.Equal(t, "DOMAIN:evil.com", RuleMatchKey(RuleTypeDomain, "https://Evil.COM:443/path"))
	require.Equal(t, "DOMAIN:xn--caf-dma.com", RuleMatchKey(RuleTypeDomain, "café.com"))
}

// TestRuleMatchKey_Tag pins TAG normalization: `#` is stripped and the value is tokenized, so
// "#UsPol", "uspol", and "us pol" all key alike.
func TestRuleMatchKey_Tag(t *testing.T) {
	require.Equal(t, "TAG:uspol", RuleMatchKey(RuleTypeTag, "#UsPol"))
	require.Equal(t, "TAG:uspol", RuleMatchKey(RuleTypeTag, "uspol"))
}

// TestRuleMatchKey_Empty confirms an unknown Type or an empty Trigger yields the empty key, which
// matches nothing -- a rule never silently degrades to matching everything.
func TestRuleMatchKey_Empty(t *testing.T) {
	require.Equal(t, "", RuleMatchKey("SOMETHING-NEW", "value"))
	require.Equal(t, "", RuleMatchKey(RuleTypeTag, ""))
	require.Equal(t, "", RuleMatchKey(RuleTypeDomain, ""))
}

// TestActorMatchKeys pins the wire gate's key set: the ACTOR key plus a DOMAIN key for every host
// suffix, and NO tag keys (the gate blocks by who, never by content). A blocked actor rule and a
// blocked domain rule must both be reachable from these keys.
func TestActorMatchKeys(t *testing.T) {
	keys := ActorMatchKeys("https://sub.example.com/@bob")

	require.Contains(t, keys, "ACTOR:https://sub.example.com/@bob")
	require.Contains(t, keys, "DOMAIN:sub.example.com")
	require.Contains(t, keys, "DOMAIN:example.com")
	require.Contains(t, keys, "DOMAIN:com")

	// No TAG keys ever come from the actor gate.
	for _, key := range keys {
		require.False(t, strings.HasPrefix(key, "TAG:"), key)
	}

	// The suffix boundary holds: notevil.com is not a suffix of evil.com.
	require.NotContains(t, ActorMatchKeys("https://notevil.com/@x"), "DOMAIN:evil.com")

	// An empty actor contributes nothing.
	require.Empty(t, ActorMatchKeys(""))
}

// TestActorMatchKeys_HostCase confirms the host is lower-cased on both the ACTOR and DOMAIN keys, so
// a mixed-case delivering actor cannot alias past a block.
func TestActorMatchKeys_HostCase(t *testing.T) {
	keys := ActorMatchKeys("https://Example.COM/@Bob")
	require.Contains(t, keys, "ACTOR:https://example.com/@Bob")
	require.Contains(t, keys, "DOMAIN:example.com")
}

// TestDomainMatchKeys pins the keyId-host key set: DOMAIN keys only, one per suffix, Punycode'd. This
// is what extends a DOMAIN block to the delivering server named in a signature's keyId.
func TestDomainMatchKeys(t *testing.T) {
	keys := DomainMatchKeys("https://relay.example.com/actor#main-key")
	require.Contains(t, keys, "DOMAIN:relay.example.com")
	require.Contains(t, keys, "DOMAIN:example.com")
	require.Empty(t, DomainMatchKeys(""))
}

// TestDocumentMatchKeys_Actor confirms an actor contributes its own ACTOR key plus a DOMAIN key for
// every suffix of its host -- which is how a DOMAIN block reaches an actor on a subdomain.
func TestDocumentMatchKeys_Actor(t *testing.T) {

	document := streams.NewDocument(mapof.Any{
		vocab.PropertyActor: "https://mastodon.social/@alice",
	})

	keys := DocumentMatchKeys(document)

	require.Contains(t, keys, "ACTOR:https://mastodon.social/@alice")
	require.Contains(t, keys, "DOMAIN:mastodon.social")
	require.Contains(t, keys, "DOMAIN:social")
}

// TestDocumentMatchKeys_AttributedTo confirms an Object's author (`attributedTo`) contributes ACTOR
// and DOMAIN keys just like an Activity's `actor` -- Objects name their author with attributedTo, so
// an ACTOR/DOMAIN rule must reach a document that carries no `actor` property at all.
func TestDocumentMatchKeys_AttributedTo(t *testing.T) {

	// A bare-string attributedTo (the common wire form)
	document := streams.NewDocument(mapof.Any{
		vocab.PropertyAttributedTo: "https://mastodon.social/@alice",
	})

	keys := DocumentMatchKeys(document)

	require.Contains(t, keys, "ACTOR:https://mastodon.social/@alice")
	require.Contains(t, keys, "DOMAIN:mastodon.social")

	// An object-valued attributedTo keys off its `id`
	document = streams.NewDocument(mapof.Any{
		vocab.PropertyAttributedTo: mapof.Any{vocab.PropertyID: "https://mastodon.social/@bob"},
	})

	require.Contains(t, DocumentMatchKeys(document), "ACTOR:https://mastodon.social/@bob")
}

// TestDocumentMatchKeys_ActorAndAuthor confirms a document carrying BOTH identities keys them both --
// and that a self-attributed document (actor == attributedTo) does not duplicate its keys.
func TestDocumentMatchKeys_ActorAndAuthor(t *testing.T) {

	// Distinct identities: both are keyed.
	document := streams.NewDocument(mapof.Any{
		vocab.PropertyActor:        "https://relay.example/@booster",
		vocab.PropertyAttributedTo: "https://origin.example/@author",
	})

	keys := DocumentMatchKeys(document)

	require.Contains(t, keys, "ACTOR:https://relay.example/@booster")
	require.Contains(t, keys, "ACTOR:https://origin.example/@author")

	// Identical identities: keyed exactly once.
	document = streams.NewDocument(mapof.Any{
		vocab.PropertyActor:        "https://origin.example/@author",
		vocab.PropertyAttributedTo: "https://origin.example/@author",
	})

	actorKeyCount := 0
	for _, key := range DocumentMatchKeys(document) {
		if key == "ACTOR:https://origin.example/@author" {
			actorKeyCount++
		}
	}

	require.Equal(t, 1, actorKeyCount)
}

// TestDocumentMatchKeys_HashtagsOnly confirms Hashtags produce TAG keys while Mentions do not --
// the guard that keeps a TAG rule for "alice" from matching a post that mentions @alice.
func TestDocumentMatchKeys_HashtagsOnly(t *testing.T) {

	document := streams.NewDocument(mapof.Any{
		vocab.PropertyActor: "https://example.com/@bob",
		vocab.PropertyTag: []any{
			mapof.Any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#uspol"},
			mapof.Any{vocab.PropertyType: vocab.LinkTypeMention, vocab.PropertyName: "alice"},
		},
	})

	keys := DocumentMatchKeys(document)

	require.Contains(t, keys, "TAG:uspol")
	require.NotContains(t, keys, "TAG:alice")
}

// TestMatchKey_RoundTrip is the load-bearing test: for each type, a Rule's key must appear among the
// keys of a document it should match, across exactly the variations that fail silently -- host case
// (ACTOR), subdomain and IDN (DOMAIN), and the `#` prefix (TAG). If the two producers ever disagree,
// a block quietly stops blocking, and this is what catches it.
func TestMatchKey_RoundTrip(t *testing.T) {

	// ACTOR: a mixed-case rule trigger matches a lower-case wire actor.
	{
		document := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://example.com/@bob"})
		require.Contains(t, DocumentMatchKeys(document), RuleMatchKey(RuleTypeActor, "https://Example.com/@bob"))
	}

	// DOMAIN: a rule on the registrable domain matches an actor on a subdomain...
	{
		document := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://sub.evil.com/@spammer"})
		require.Contains(t, DocumentMatchKeys(document), RuleMatchKey(RuleTypeDomain, "evil.com"))
	}

	// ...but NOT a look-alike domain that merely ends in the same string.
	{
		document := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://notevil.com/@innocent"})
		require.NotContains(t, DocumentMatchKeys(document), RuleMatchKey(RuleTypeDomain, "evil.com"))
	}

	// DOMAIN (IDN): a Unicode rule matches the Punycode wire host.
	{
		document := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://xn--caf-dma.com/@x"})
		require.Contains(t, DocumentMatchKeys(document), RuleMatchKey(RuleTypeDomain, "café.com"))
	}

	// TAG: a bare rule trigger matches a `#`-prefixed wire hashtag.
	{
		document := streams.NewDocument(mapof.Any{
			vocab.PropertyActor: "https://example.com/@bob",
			vocab.PropertyTag:   mapof.Any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#uspol"},
		})
		require.Contains(t, DocumentMatchKeys(document), RuleMatchKey(RuleTypeTag, "uspol"))
	}
}
