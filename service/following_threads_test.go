package service

import (
	"strconv"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newTestWalk builds a primaryPostWalk backed by the given rule store (empty = nothing filtered).
func newTestWalk(store *ruleStore, userID primitive.ObjectID) *primaryPostWalk {
	service, session := newRuleService(store)
	return &primaryPostWalk{
		ruleService: service,
		session:     session,
		userID:      userID,
		now:         dispositionNow,
		seen:        make(map[string]bool),
	}
}

// A lone document with no rules is its own primary post.
func TestPrimaryPost_Primary(t *testing.T) {

	walk := newTestWalk(&ruleStore{}, primitive.NewObjectID())

	original := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://document-1.com/",
	})

	primary, originType, dropped, err := walk.primaryPost(original, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.False(t, dropped)
	require.Equal(t, model.OriginTypePrimary, originType)
	require.Equal(t, "https://document-1.com/", primary.ID())
}

// A reply resolves UP to its (inline) parent, and the origin type becomes REPLY.
func TestPrimaryPost_Reply(t *testing.T) {

	walk := newTestWalk(&ruleStore{}, primitive.NewObjectID())

	original := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://document-1.com/",
		vocab.PropertyInReplyTo: map[string]any{
			vocab.PropertyID: "https://document-2.com/",
		},
	})

	primary, originType, dropped, err := walk.primaryPost(original, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.False(t, dropped)
	require.Equal(t, model.OriginTypeReply, originType)
	require.Equal(t, "https://document-2.com/", primary.ID())
}

// A document whose author is BLOCKED is dropped, so it creates no newsfeed item.
func TestPrimaryPost_BlockedAuthorDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://spammer.example/@evil")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{block}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://post.example/1",
		vocab.PropertyAttributedTo: "https://spammer.example/@evil",
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A MUTED author is dropped too -- unlike the wire gate, the newsfeed filters on BLOCK and MUTE (R18).
func TestPrimaryPost_MutedAuthorDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeActor, model.RuleActionMute, "https://noisy.example/@chatty")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://post.example/1",
		vocab.PropertyAttributedTo: "https://noisy.example/@chatty",
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A blocked author in the PARENT (an ancestor) drops the whole item -- the `dropped` verdict must
// propagate down the return chain, not get swallowed by the child returning itself.
func TestPrimaryPost_BlockedAncestorPropagates(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://spammer.example/@evil")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{block}}, userID)

	reply := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/reply",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		vocab.PropertyInReplyTo: map[string]any{
			vocab.PropertyID:           "https://good.example/parent",
			vocab.PropertyAttributedTo: "https://spammer.example/@evil",
		},
	})

	_, _, dropped, err := walk.primaryPost(reply, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// An Announce from a BLOCKED booster is dropped BEFORE unwrapping -- even though the boosted post's
// own author is clean, a blocked booster's boost must create nothing (R2).
func TestPrimaryPost_BlockedBoosterDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://booster.example/@loud")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{block}}, userID)

	announce := streams.NewDocument(map[string]any{
		vocab.PropertyType:  vocab.ActivityTypeAnnounce,
		vocab.PropertyActor: "https://booster.example/@loud",
		vocab.PropertyObject: map[string]any{
			vocab.PropertyID:           "https://post.example/1",
			vocab.PropertyAttributedTo: "https://clean.example/@ok",
		},
	})

	_, _, dropped, err := walk.primaryPost(announce, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A DOMAIN block on a parent's host stops the walk at the pre-fetch check, dropping the reply.
func TestPrimaryPost_BlockedParentHostDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeDomain, model.RuleActionBlock, "blocked.example")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{block}}, userID)

	reply := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://good.example/reply",
		vocab.PropertyInReplyTo: map[string]any{
			vocab.PropertyID: "https://blocked.example/parent",
		},
	})

	_, _, dropped, err := walk.primaryPost(reply, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A post that QUOTES content on a blocked DOMAIN is dropped (R18), caught at the host check with no
// fetch. The quote rides the Misskey-style `quoteUrl` field.
func TestPrimaryPost_QuotedBlockedDomainDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	block := summaryRule(userID, model.RuleTypeDomain, model.RuleActionBlock, "blocked.example")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{block}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/1",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		"quoteUrl":                 "https://blocked.example/nasty-post",
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// An inReplyTo chain deeper than maxReplyDepth stops gracefully: it is NOT dropped (a legit deep
// thread still creates an item), but the walk stops climbing at the limit -- the DoS backstop.
func TestPrimaryPost_DepthLimit(t *testing.T) {

	walk := newTestWalk(&ruleStore{}, primitive.NewObjectID())

	// Build an inline reply chain deeper than the limit.
	var build func(n int) map[string]any
	build = func(n int) map[string]any {
		node := map[string]any{vocab.PropertyID: "https://x.example/" + strconv.Itoa(n)}
		if n > 0 {
			node[vocab.PropertyInReplyTo] = build(n - 1)
		}
		return node
	}

	deep := streams.NewDocument(build(maxReplyDepth + 5))

	primary, _, dropped, err := walk.primaryPost(deep, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.False(t, dropped)
	require.True(t, primary.NotNil())
}

// A TAG MUTE drops a post carrying the hashtag -- content filtering at ingest (D12), not just
// identity filtering. Rule trigger and wire tag both carry the `#` prefix (the Mastodon convention).
func TestPrimaryPost_MutedTagDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/tagged",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		vocab.PropertyTag: map[string]any{
			vocab.PropertyType: vocab.LinkTypeHashtag,
			vocab.PropertyName: "#qatest",
		},
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// The control post WITHOUT the hashtag still lands -- a TAG rule matches the tag, never the author.
func TestPrimaryPost_ControlPostWithoutTagKept(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/control",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
	})

	primary, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.False(t, dropped)
	require.Equal(t, "https://good.example/control", primary.ID())
}

// A domain-wide ADMIN TAG rule (UserID zero) drops the tagged post for a regular User -- the
// server-wide moderation case from the QA report.
func TestPrimaryPost_AdminTagRuleDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	adminMute := summaryRule(primitive.NilObjectID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{adminMute}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/tagged",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		vocab.PropertyTag: map[string]any{
			vocab.PropertyType: vocab.LinkTypeHashtag,
			vocab.PropertyName: "#qatest",
		},
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A clean booster Announcing a TAGGED post is dropped -- the muted tag rides the boosted object, and
// the walk checks every Object node it unwraps.
func TestPrimaryPost_MutedTagOnBoostedPostDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	announce := streams.NewDocument(map[string]any{
		vocab.PropertyType:  vocab.ActivityTypeAnnounce,
		vocab.PropertyActor: "https://booster.example/@clean",
		vocab.PropertyObject: map[string]any{
			vocab.PropertyID:           "https://origin.example/tagged",
			vocab.PropertyAttributedTo: "https://origin.example/@author",
			vocab.PropertyTag: map[string]any{
				vocab.PropertyType: vocab.LinkTypeHashtag,
				vocab.PropertyName: "#qatest",
			},
		},
	})

	_, _, dropped, err := walk.primaryPost(announce, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A muted hashtag on an ANCESTOR drops the reply's item too, matching the identity semantics: any
// filtered node in the provenance chain stops the walk (R18).
func TestPrimaryPost_MutedTagUpthreadDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	reply := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/reply",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		vocab.PropertyInReplyTo: map[string]any{
			vocab.PropertyID:           "https://good.example/parent",
			vocab.PropertyAttributedTo: "https://good.example/@op",
			vocab.PropertyTag: map[string]any{
				vocab.PropertyType: vocab.LinkTypeHashtag,
				vocab.PropertyName: "#qatest",
			},
		},
	})

	_, _, dropped, err := walk.primaryPost(reply, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.True(t, dropped)
}

// A Mention-typed tag whose name matches the trigger does NOT drop the post -- TAG rules match
// Hashtags only (D12), so a rule for "qatest" cannot silence posts that merely mention @qatest.
func TestPrimaryPost_MentionTagNotDropped(t *testing.T) {

	userID := primitive.NewObjectID()
	mute := summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "#qatest")
	walk := newTestWalk(&ruleStore{records: []model.RuleSummary{mute}}, userID)

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:           "https://good.example/mention",
		vocab.PropertyAttributedTo: "https://good.example/@friend",
		vocab.PropertyTag: map[string]any{
			vocab.PropertyType: vocab.LinkTypeMention,
			vocab.PropertyName: "qatest",
		},
	})

	_, _, dropped, err := walk.primaryPost(document, model.OriginTypePrimary, 0)

	require.Nil(t, err)
	require.False(t, dropped)
}
