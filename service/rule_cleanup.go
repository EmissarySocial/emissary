package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// previousRuleState returns the row currently stored for the provided RuleID, or a zero-value
// Rule for a fresh insert. Loaded explicitly because nothing else on the Save path exposes the
// pre-mutation row -- the cleanup trigger reads its Action/MatchKey, and P7-2 reads its
// PublishedAction.
func (service *Rule) previousRuleState(session data.Session, ruleID primitive.ObjectID) model.Rule {

	const location = "service.Rule.previousRuleState"

	previous := model.Rule{}

	if err := service.Load(session, exp.Equal("_id", ruleID), &previous); err != nil {

		// A missing row is simply a fresh insert; anything else is reported but treated the
		// same way -- a missed transition is caught by the render-time backstop, never by a 500.
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Loading previous Rule state; treating as new", ruleID))
		}

		return model.Rule{}
	}

	return previous
}

// cleanupTransitions is the PURE decision half of enqueueCleanup: given the stored row's prior
// state and the incoming state, it reports which retroactive passes the change requires (R8) --
// purgeAll on transitions into BLOCK, purgeNewsfeed on transitions into MUTE (D1), restore on
// transitions out of BLOCK. An edited trigger re-aims a live rule, so it fires the same passes
// in both directions; the restore pass needs no old key because it re-evaluates every paused row
// against the REMAINING rules.
func cleanupTransitions(oldAction string, oldMatchKey string, newAction string, newMatchKey string) (purgeAll bool, purgeNewsfeed bool, restore bool) {

	purgeAll = (newAction == model.RuleActionBlock) && (oldAction != model.RuleActionBlock)
	restore = (oldAction == model.RuleActionBlock) && (newAction != model.RuleActionBlock)
	purgeNewsfeed = (newAction == model.RuleActionMute) && (oldAction != model.RuleActionMute)

	if (oldMatchKey != "") && (oldMatchKey != newMatchKey) {
		purgeAll = purgeAll || (newAction == model.RuleActionBlock)
		restore = restore || (oldAction == model.RuleActionBlock)
		purgeNewsfeed = purgeNewsfeed || (newAction == model.RuleActionMute)
	}

	return purgeAll, purgeNewsfeed, restore
}

// enqueueCleanup publishes a post-commit "Rule-Cleanup" task when a Rule change requires
// retroactive work (R8). LABEL churn and no-op edits enqueue nothing: labels are derived at
// render time.
//
// RULE: the payload is a SNAPSHOT, never a ruleID -- under hard delete (D16) the row is gone
// before the post-commit task runs, so a load-by-ID would silently no-op and purge nothing.
func (service *Rule) enqueueCleanup(session data.Session, rule model.Rule, oldAction string, oldMatchKey string, newAction string) {

	purgeAll, purgeNewsfeed, restore := cleanupTransitions(oldAction, oldMatchKey, newAction, rule.MatchKey)

	if !purgeAll && !purgeNewsfeed && !restore {
		return
	}

	postcommit.Publish(session, service.queue, "Rule-Cleanup", mapof.Any{
		"hostname":      uri.Hostname(service.host),
		"userId":        rule.UserID.Hex(),
		"type":          rule.Type,
		"matchKey":      rule.MatchKey,
		"followingId":   rule.FollowingID.Hex(),
		"purgeAll":      purgeAll,
		"purgeNewsfeed": purgeNewsfeed,
		"restore":       restore,
	})
}
