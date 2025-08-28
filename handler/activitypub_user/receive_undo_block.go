package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeUndo, vocab.ActivityTypeBlock, func(context Context, activity streams.Document) error {

		const location = "handler.activitypub_user.receiveUndoBlock"

		// Verify that this message comes from a valid "Following" object.
		followingService := context.factory.Following()
		following := model.NewFollowing()

		// If the "Following" record cannot be found, then halt
		if err := followingService.LoadByURL(context.session, context.user.UserID, activity.Actor().ID(), &following); err != nil {
			return nil
		}

		// If the User is not listening to Rules from this source, then halt.
		if following.RuleAction == model.FollowingRuleActionIgnore {
			return nil
		}

		// Try to find a Rule that matches this Activity
		ruleService := context.factory.Rule()
		rule := ruleFromActivity(&following, activity.Object())

		if err := ruleService.LoadByFollowing(context.session, context.user.UserID, following.FollowingID, rule.Type, rule.Trigger, &rule); err != nil {
			if derp.IsNotFound(err) {
				return nil
			}
			return derp.Wrap(err, location, "Error loading rule", activity.Value())
		}

		// Remove the Rule
		if err := ruleService.Delete(context.session, &rule, "Removed via ActivityPub"); err != nil {
			return derp.Wrap(err, location, "Error deleting rule", activity.Value())
		}

		// Success.
		return nil
	})
}
