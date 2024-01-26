package activitypub

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/davecgh/go-spew/spew"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeBlock, vocab.Any, func(factory *domain.Factory, user *model.User, activity streams.Document) error {

		const location = "handler.activitypub.receiveBlock"

		// Verify that this message comes from a valid "Following" object.
		followingService := factory.Following()
		following := model.NewFollowing()

		// If the "Following" record cannot be found, then halt
		if err := followingService.LoadByURL(user.UserID, activity.Actor().ID(), &following); err != nil {
			return nil
		}

		// If the user is not listening to rules from this source, then halt.
		if following.RuleAction == model.FollowingRuleActionIgnore {
			return nil
		}

		// Create a new Rule using the information from the received Activity
		object := activity.Object()

		rule := model.NewRule()
		rule.UserID = following.UserID
		rule.FollowingID = following.FollowingID
		rule.Type = model.RuleTypeActor // default value
		rule.Action = following.RuleAction
		rule.Label = "Blocked by " + activity.Actor().Name()
		rule.PublishDate = activity.Published().Unix()
		rule.Summary = object.Summary()

		spew.Dump("Object Type", object.Type())

		switch object.Type() {

		// Domain Blocks are represented as Applications, Services, or Organizations
		case vocab.ActorTypeApplication, vocab.ActorTypeService, vocab.ActorTypeOrganization:
			rule.Type = model.RuleTypeDomain
			rule.Trigger = first.String(object.URL(), object.ID())
			spew.Dump("DOMAIN BLOCK")

		// Content Blocks are represented as Notes
		case vocab.ObjectTypeNote:
			rule.Type = model.RuleTypeContent
			rule.Trigger = object.Content()
			spew.Dump("CONTENT BLOCK")

		// Anything else (incl. Null) is treated as an Actor Block
		default:
			rule.Type = model.RuleTypeActor
			rule.Trigger = object.ID()
			spew.Dump("ACTOR BLOCK")
		}

		spew.Dump("FINAL RULE VALUE: ", rule)

		// Try to save the new rule to the database (with de-duplication)
		if err := factory.Rule().Save(&rule, "Received via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Error saving rule", activity.Value(), rule)
		}

		// Success.
		return nil
	})
}
