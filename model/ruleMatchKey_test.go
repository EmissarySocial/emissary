package model

import (
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
