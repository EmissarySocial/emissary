package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeBlock, vocab.Any, inbox_BlockAny)
}

func inbox_BlockAny(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_BlockAny"

	// Verify that this message comes from a valid "Following" object.
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then halt
	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.Actor().ID(), &following); err != nil {
		return nil
	}

	// If the user is not listening to rules from this source, then halt.
	if following.RuleAction == model.FollowingRuleActionIgnore {
		return nil
	}

	// Create a new Rule using the information from the received Activity
	rule := ruleFromActivity(&following, activity)

	// Try to save the new rule to the database (with de-duplication)
	if err := context.factory.Rule().Save(context.session, &rule, "Received via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Unable to save rule", activity.Value(), rule)
	}

	// Success.
	return nil
}

func ruleFromActivity(following *model.Following, activity streams.Document) model.Rule {

	object := activity.Object()

	result := model.NewRule()
	result.UserID = following.UserID
	result.FollowingID = following.FollowingID
	result.FollowingLabel = following.Label
	result.Type = model.RuleTypeActor // default value
	result.Action = following.RuleAction
	result.Label = "Blocked by " + activity.Actor().Name()
	result.PublishDate = activity.Published().Unix()
	result.Summary = object.Summary()

	switch object.Type() {

	// Domain Blocks are represented as Applications, Services, or Organizations
	case vocab.ActorTypeApplication, vocab.ActorTypeService, vocab.ActorTypeOrganization:
		result.Type = model.RuleTypeDomain
		result.Trigger = first.String(object.URL(), object.ID())

	// Content Blocks are represented as Notes
	case vocab.ObjectTypeNote:
		result.Type = model.RuleTypeContent
		result.Trigger = object.Content()

	// Anything else (incl. Null) is treated as an Actor Block
	default:
		result.Type = model.RuleTypeActor
		result.Trigger = object.ID()
	}

	return result
}
