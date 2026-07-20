package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRule_LabelDocuments pins the render-path stamp: hidden verdicts and annotations land in
// each document's Metadata.Labels, clean documents stay unlabeled, and USER-tier labels carry
// their attribution Href.
func TestRule_LabelDocuments(t *testing.T) {

	userID := primitive.NewObjectID()

	blockRule := summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer")
	labelRule := summaryRule(userID, model.RuleTypeActor, model.RuleActionLabel, "https://noisy.example/@chatty")
	labelRule.Label = "Chronic Reply Guy"

	store := &ruleStore{records: []model.RuleSummary{blockRule, labelRule}}
	service, session := newRuleService(store)

	documents := []streams.Document{
		streams.NewDocument(map[string]any{"id": "https://evil.example/notes/1", "actor": "https://evil.example/@spammer"}),
		streams.NewDocument(map[string]any{"id": "https://noisy.example/notes/2", "actor": "https://noisy.example/@chatty"}),
		streams.NewDocument(map[string]any{"id": "https://friendly.example/notes/3", "actor": "https://friendly.example/@pal"}),
	}

	service.LabelDocuments(session, userID, documents)

	// The blocked actor's document is hidden, with a link to the user's own rule
	require.True(t, documents[0].Metadata.Labels.IsHidden())
	require.Equal(t, "Blocked by your rules", documents[0].Metadata.Labels.Reason())
	require.Equal(t, "/@me/settings/rule-edit?ruleId="+blockRule.RuleID.Hex(), documents[0].Metadata.Labels.Hidden()[0].Href)

	// The labeled actor's document is annotated but NOT hidden
	require.False(t, documents[1].Metadata.Labels.IsHidden())
	require.True(t, documents[1].Metadata.Labels.HasAnnotations())
	require.Equal(t, "Chronic Reply Guy", documents[1].Metadata.Labels.Annotations()[0].Value)

	// The clean document carries no labels at all
	require.Empty(t, documents[2].Metadata.Labels)
}

// TestRule_LabelDocuments_AdminTier pins that ADMIN-tier labels attribute in text but never
// link: "server policy" is not user-removable (D8/D9).
func TestRule_LabelDocuments_AdminTier(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(primitive.NilObjectID, model.RuleTypeDomain, model.RuleActionBlock, "banned.example"),
	}}

	service, session := newRuleService(store)

	documents := []streams.Document{
		streams.NewDocument(map[string]any{"id": "https://banned.example/notes/1", "actor": "https://banned.example/@anyone"}),
	}

	service.LabelDocuments(session, userID, documents)

	require.True(t, documents[0].Metadata.Labels.IsHidden())
	require.Equal(t, "Blocked by server policy", documents[0].Metadata.Labels.Reason())
	require.Equal(t, "", documents[0].Metadata.Labels.Hidden()[0].Href)
}

// TestRule_LabelNotifications pins the notification stamp: verdicts derive from the snapshotted
// Actor at render time (R8: derive, don't record).
func TestRule_LabelNotifications(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer"),
	}}

	service, session := newRuleService(store)

	notifications := []model.Notification{
		{Actor: model.PersonLink{ProfileURL: "https://evil.example/@spammer"}},
		{Actor: model.PersonLink{ProfileURL: "https://friendly.example/@pal"}},
	}

	service.LabelNotifications(session, userID, notifications)

	require.True(t, notifications[0].Labels.IsHidden())
	require.Empty(t, notifications[1].Labels)
}

// TestRule_LabelSearchResults pins the search stamp: verdicts derive from the result's URL and
// tags; hidden results are dropped by the SearchBuilder, so only the stamp is pinned here.
func TestRule_LabelSearchResults(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, "https://evil.example/@spammer"),
		summaryRule(userID, model.RuleTypeTag, model.RuleActionMute, "crypto"),
	}}

	service, session := newRuleService(store)

	results := []model.SearchResult{
		{URL: "https://evil.example/@spammer"},
		{URL: "https://friendly.example/notes/9", Tags: []string{"crypto"}},
		{URL: "https://friendly.example/@pal"},
	}

	service.LabelSearchResults(session, userID, results)

	require.True(t, results[0].Labels.IsHidden())
	require.True(t, results[1].Labels.IsHidden())
	require.Empty(t, results[2].Labels)
}
