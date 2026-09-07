package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mailing List Sync
 *
 * These read the post-commit spool, which is what the two hooks actually
 * write to. What matters is not that a task is well-formed but WHICH saves
 * and deletes decide to enqueue one -- because both wrong answers are silent,
 * and one of them is destructive at a third party. See MAILING-LISTS.md 1.2.
 ******************************************/

// newMailingListFollower returns the kind of Follower that a mailing list mirrors
func newMailingListFollower() model.Follower {

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.StateID = model.FollowerStateActive
	follower.Method = model.FollowerMethodEmail
	follower.Format = model.MimeTypeHTML
	follower.Actor.Name = "Sarah Connor"
	follower.Actor.ProfileURL = "sarah@connor.mil"
	follower.Actor.EmailAddress = "sarah@connor.mil"

	return follower
}

// TestFollower_SaveEnqueuesTheAdd walks the happy path of a confirmed subscription
func TestFollower_SaveEnqueuesTheAdd(t *testing.T) {

	service, session, tasks := newSpooledFollowerService()
	follower := newMailingListFollower()

	require.NoError(t, service.Save(session, &follower, "Confirmed"))

	spooled := tasks.Drain()
	require.Len(t, spooled, 1)
	require.Equal(t, MailingListAddMember, spooled[0].Name)
	require.Equal(t, follower.FollowerID, spooled[0].Arguments["followerId"])
}

// TestFollower_SaveEnqueuesNothingForEveryoneElse pins the four conditions on the add hook
func TestFollower_SaveEnqueuesNothingForEveryoneElse(t *testing.T) {

	// Each of these is a real record that exists in production, and pushing any of them to a
	// third party would be wrong in its own way: an unconfirmed address nobody agreed to
	// share, a person the User blocked, an ActivityPub actor who never gave an address, or a
	// search-alert subscriber who is not a newsletter subscriber at all.

	tests := map[string]func(*model.Follower){
		"unconfirmed":        func(f *model.Follower) { f.StateID = model.FollowerStatePending },
		"blocked by a rule":  func(f *model.Follower) { f.StateID = model.FollowerStatePaused },
		"activitypub":        func(f *model.Follower) { f.Method = model.FollowerMethodActivityPub },
		"follows a stream":   func(f *model.Follower) { f.ParentType = model.FollowerTypeStream },
		"follows a search":   func(f *model.Follower) { f.ParentType = model.FollowerTypeSearch },
		"has no address":     func(f *model.Follower) { f.Actor.EmailAddress = "" },
		"import placeholder": func(f *model.Follower) { f.StateID = model.FollowerStateImportPending },
	}

	for name, mutate := range tests {

		t.Run(name, func(t *testing.T) {

			service, session, tasks := newSpooledFollowerService()

			follower := newMailingListFollower()
			mutate(&follower)

			// Save validates, so a record it refuses proves nothing either way
			_ = service.Save(session, &follower, "Saved")

			require.Empty(t, tasks.Drain(), "this Follower must not reach a mailing list")
		})
	}
}

// TestFollower_DeleteEnqueuesTheRemoveWithTheAddress guards the one value the task cannot
// look up for itself
func TestFollower_DeleteEnqueuesTheRemoveWithTheAddress(t *testing.T) {

	// By the time the task runs, this record is gone. An address left out of the arguments
	// is one that can never be recovered, and the unsubscribe silently reaches nobody.

	follower := newMailingListFollower()
	service, session, tasks := newSpooledFollowerService(follower)

	require.NoError(t, service.Delete(session, &follower, "Unsubscribed"))

	spooled := tasks.Drain()
	require.Len(t, spooled, 1)
	require.Equal(t, MailingListRemoveMember, spooled[0].Name)
	require.Equal(t, "sarah@connor.mil", spooled[0].Arguments["emailAddress"])
}

// TestFollower_DeleteByUserIDEnqueuesNothing is D14, and the regression it guards is silent
// and destructive at the far end
func TestFollower_DeleteByUserIDEnqueuesNothing(t *testing.T) {

	// Follower.Delete is the funnel every removal path reaches, and it is also where the
	// outbound unsubscribe hook lives.  So without DeleteWithoutSync, deleting ONE User
	// unsubscribes every one of their followers from that same User's own audience --
	// through the API, at Mailchimp, where an unsubscribe cannot be undone.

	// The owner's ID is left zero so that CalcFollowerCount short-circuits: it would otherwise
	// reach for a User collection, and a raw Mongo update, that this harness does not have.
	userID := primitive.NilObjectID

	first := newMailingListFollower()
	second := newMailingListFollower()
	second.Actor.EmailAddress = "kyle@reese.mil"

	service, session, tasks := newSpooledFollowerService(first, second)

	require.NoError(t, service.DeleteByUserID(session, userID, "User deleted"))

	require.Len(t, session.collection.deleted, 2, "both Followers must actually be deleted")
	require.Empty(t, tasks.Drain(), "deleting a User must not unsubscribe anyone at Mailchimp (D14)")
}

// TestFollower_DeleteWithoutSyncStillDeletes confirms the quiet path is not a no-op
func TestFollower_DeleteWithoutSyncStillDeletes(t *testing.T) {

	follower := newMailingListFollower()
	service, session, tasks := newSpooledFollowerService(follower)

	require.NoError(t, service.DeleteWithoutSync(session, &follower, "Unsubscribed at Mailchimp"))

	require.Len(t, session.collection.deleted, 1)
	require.Empty(t, tasks.Drain())
}

// TestFollower_DeleteEnqueuesNothingWhenTheDeleteFails pins the order of the two halves
func TestFollower_DeleteEnqueuesNothingWhenTheDeleteFails(t *testing.T) {

	// Outside a transaction, postcommit.Publish sends immediately. So if the unsubscribe were
	// published before the delete ran, a delete that then failed would leave a Follower who
	// still exists here and has already been unsubscribed at Mailchimp.

	follower := newMailingListFollower()
	service, session, tasks := newSpooledFollowerService(follower)
	session.collection.deleteError = derp.Internal("test", "the database is on fire")

	require.Error(t, service.Delete(session, &follower, "Unsubscribed"))
	require.Empty(t, tasks.Drain(), "a delete that failed must not have unsubscribed anyone")
}

/******************************************
 * Email address normalization
 ******************************************/

// TestFollower_SaveNormalizesTheAddress pins the one form an address is ever stored in
func TestFollower_SaveNormalizesTheAddress(t *testing.T) {

	service, session, _ := newSpooledFollowerService()

	follower := newMailingListFollower()
	follower.Actor.EmailAddress = " Sarah@Connor.MIL "
	follower.Actor.ProfileURL = "Sarah@Connor.MIL"

	require.NoError(t, service.Save(session, &follower, "Signed up"))

	require.Len(t, session.collection.saved, 1)
	require.Equal(t, "sarah@connor.mil", session.collection.saved[0].Actor.EmailAddress)
	require.Equal(t, "sarah@connor.mil", session.collection.saved[0].Actor.ProfileURL, "an EMAIL Follower carries the address in profileUrl too")
}

// TestFollower_SaveLeavesAnActivityPubProfileURLAlone guards the other half of the rule
func TestFollower_SaveLeavesAnActivityPubProfileURLAlone(t *testing.T) {

	// A URL path is case-sensitive, so lowercasing it would point at a different actor

	service, session, _ := newSpooledFollowerService()

	follower := newMailingListFollower()
	follower.Method = model.FollowerMethodActivityPub
	follower.Actor.EmailAddress = ""
	follower.Actor.ProfileURL = "https://example.com/@Sarah"

	require.NoError(t, service.Save(session, &follower, "Followed"))

	require.Len(t, session.collection.saved, 1)
	require.Equal(t, "https://example.com/@Sarah", session.collection.saved[0].Actor.ProfileURL)
}

// TestFollower_LoadByEmailAddressIgnoresCase is what lets a webhook find the row it names
func TestFollower_LoadByEmailAddressIgnoresCase(t *testing.T) {

	stored := newMailingListFollower() // sarah@connor.mil, lowercase
	service, session, _ := newSpooledFollowerService(stored)

	found := model.NewFollower()
	require.NoError(t, service.LoadByEmailAddress(session, stored.ParentID, " SARAH@Connor.mil ", &found))
	require.Equal(t, stored.FollowerID, found.FollowerID)
}
