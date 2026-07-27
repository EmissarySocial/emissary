package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests lock the R11 gate at the top of Connect. That gate is what MAKES the asrules reveal on
// the actor fetch safe: with the reveal in place, the rules client no longer refuses the fetch, so a
// BLOCK is enforced here or nowhere. Every case leaves activityService nil on purpose -- a panic
// would prove the gate let execution through to the network.

const blockedActorURL = "https://evil.example/@bob"

// TestConnect_BlockedActorRefused proves a User cannot create a Following for an actor they have
// blocked, and that the refusal is a friendly 422 rather than the opaque 403 the rules client used to
// raise (which took down the whole enclosing form).
func TestConnect_BlockedActorRefused(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeActor, model.RuleActionBlock, blockedActorURL),
	}}

	ruleService, session := newRuleService(store)
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = userID
	following.URL = blockedActorURL

	err := followingService.Connect(session, &following)

	require.NotNil(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
}

// TestConnect_AdminBlockRefused proves the gate honors ADMIN-tier rules too -- a server-policy block
// (stored against the nil UserID) still refuses the follow, so the reveal cannot be used to route
// around moderation the User does not control.
func TestConnect_AdminBlockRefused(t *testing.T) {

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(primitive.NilObjectID, model.RuleTypeActor, model.RuleActionBlock, blockedActorURL),
	}}

	ruleService, session := newRuleService(store)
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()
	following.URL = blockedActorURL

	err := followingService.Connect(session, &following)

	require.NotNil(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
}

// TestConnect_DomainBlockRefused proves the gate keys on the actor's DOMAIN as well as its URL, so a
// domain-wide block is not sidestepped by following one of its actors directly.
func TestConnect_DomainBlockRefused(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeDomain, model.RuleActionBlock, "evil.example"),
	}}

	ruleService, session := newRuleService(store)
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = userID
	following.URL = blockedActorURL

	err := followingService.Connect(session, &following)

	require.NotNil(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
}

// TestConnect_MutedActorNotRefused is the counterpart to the reveal: a MUTE hides an actor's content
// but says nothing about whether the User may follow them. Before the reveal, the rules client
// refused this fetch outright and following a muted account was impossible. The gate must let it
// through -- proven here by reaching the (nil) activityService and panicking rather than returning.
func TestConnect_MutedActorNotRefused(t *testing.T) {

	userID := primitive.NewObjectID()

	store := &ruleStore{records: []model.RuleSummary{
		summaryRule(userID, model.RuleTypeActor, model.RuleActionMute, blockedActorURL),
	}}

	ruleService, session := newRuleService(store)
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = userID
	following.URL = blockedActorURL

	require.Panics(t, func() {
		_ = followingService.Connect(session, &following) //nolint:errcheck // the panic IS the assertion
	})
}

// TestConnect_NoRulesNotRefused proves the gate is not a blanket refusal: with no rules at all, Connect
// proceeds to the actor fetch (again reaching the nil activityService).
func TestConnect_NoRulesNotRefused(t *testing.T) {

	ruleService, session := newRuleService(&ruleStore{})
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()
	following.URL = blockedActorURL

	require.Panics(t, func() {
		_ = followingService.Connect(session, &following) //nolint:errcheck // the panic IS the assertion
	})
}

// TestConnect_AlreadyActivityPubShortCircuits proves the existing "already connected" guard still runs
// FIRST -- the new gate must not add a rules query to a call that returns immediately.
func TestConnect_AlreadyActivityPubShortCircuits(t *testing.T) {

	ruleService, session := newRuleService(&ruleStore{})
	followingService := Following{ruleService: ruleService}

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()
	following.URL = blockedActorURL
	following.Method = model.FollowingMethodActivityPub

	require.Nil(t, followingService.Connect(session, &following))
}
