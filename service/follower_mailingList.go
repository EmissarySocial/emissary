package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
)

/******************************************
 * Mailing List Sync
 *
 * The two places a Follower change becomes a Mailchimp change. Both enqueue
 * rather than call: a settings save must not wait on a third party, and must
 * not fail because one is down. See MAILING-LISTS.md 1.2.
 ******************************************/

// MailingListAddMember is the queue task that writes a Follower into a User's mailing list
const MailingListAddMember = "MailingList-AddMember"

// MailingListRemoveMember is the queue task that unsubscribes an address from a User's mailing list
const MailingListRemoveMember = "MailingList-RemoveMember"

// publishMailingListAdd enqueues the outbound push for a Follower who should be on the
// User's mailing list
func (service *Follower) publishMailingListAdd(session data.Session, follower *model.Follower) {

	// TEMPORARY (Mailchimp sync diagnosis): remove once the sync is confirmed working
	log.Info().
		Str("trace", "MailchimpTrace").
		Str("step", "1-publishAdd:enter").
		Str("followerId", follower.FollowerID.Hex()).
		Str("parentId", follower.ParentID.Hex()).
		Str("parentType", follower.ParentType).
		Str("method", follower.Method).
		Str("stateId", follower.StateID).
		Str("emailAddress", follower.Actor.EmailAddress).
		Str("host", service.host).
		Str("hostname", uri.Hostname(service.host)).
		Bool("queueIsNil", service.queue == nil).
		Bool("sessionIsNil", session == nil).
		Msg("MailchimpTrace: Follower.Save reached publishMailingListAdd")

	if !isMailingListFollower(follower) {
		log.Info().Str("trace", "MailchimpTrace").Str("step", "1-publishAdd:skip").Str("followerId", follower.FollowerID.Hex()).
			Str("reason", "not a mailing-list follower (parentType must be User, method must be EMAIL, email must be non-empty)").
			Msg("MailchimpTrace: NOT enqueuing MailingList-AddMember")
		return
	}

	// RULE: only ACTIVE. An EMAIL Follower is PENDING until they click the link in the
	// confirmation email, and pushing before that would hand a third party an address
	// nobody has confirmed (D5).
	if follower.StateID != model.FollowerStateActive {
		log.Info().Str("trace", "MailchimpTrace").Str("step", "1-publishAdd:skip").Str("followerId", follower.FollowerID.Hex()).
			Str("stateId", follower.StateID).Str("reason", "follower is not ACTIVE").
			Msg("MailchimpTrace: NOT enqueuing MailingList-AddMember")
		return
	}

	log.Info().Str("trace", "MailchimpTrace").Str("step", "1-publishAdd:publish").Str("followerId", follower.FollowerID.Hex()).
		Str("userId", follower.ParentID.Hex()).Str("hostname", uri.Hostname(service.host)).
		Msg("MailchimpTrace: enqueuing MailingList-AddMember via postcommit.Publish")

	// Save cannot see the previous state, so this fires on every save of a matching record.
	// That is fine: the member call is an upsert, so a repeat writes the same values.
	postcommit.Publish(
		session,
		service.queue,
		MailingListAddMember,
		mapof.Any{
			"hostname":   uri.Hostname(service.host),
			"userId":     follower.ParentID,
			"followerId": follower.FollowerID,
		},
	)
}

// publishMailingListRemove enqueues the outbound unsubscribe for a Follower who is going away
func (service *Follower) publishMailingListRemove(session data.Session, follower *model.Follower) {

	// TEMPORARY (Mailchimp sync diagnosis): remove once the sync is confirmed working
	log.Info().
		Str("trace", "MailchimpTrace").
		Str("step", "1-publishRemove:enter").
		Str("followerId", follower.FollowerID.Hex()).
		Str("parentId", follower.ParentID.Hex()).
		Str("parentType", follower.ParentType).
		Str("method", follower.Method).
		Str("emailAddress", follower.Actor.EmailAddress).
		Bool("isMailingListFollower", isMailingListFollower(follower)).
		Bool("queueIsNil", service.queue == nil).
		Msg("MailchimpTrace: Follower.Delete reached publishMailingListRemove")

	if !isMailingListFollower(follower) {
		return
	}

	// RULE: the address travels in the arguments. By the time this task runs the Follower is
	// deleted, so nothing can look it up -- and a task that cannot name its member would
	// unsubscribe nobody, silently.
	postcommit.Publish(
		session,
		service.queue,
		MailingListRemoveMember,
		mapof.Any{
			"hostname":     uri.Hostname(service.host),
			"userId":       follower.ParentID,
			"followerId":   follower.FollowerID,
			"emailAddress": follower.Actor.EmailAddress,
		},
	)
}

// isMailingListFollower returns TRUE if this Follower is the kind that a mailing list mirrors
func isMailingListFollower(follower *model.Follower) bool {

	// A search-alert subscriber is not a newsletter subscriber, and the Mailchimp namespace
	// hangs off a User -- which Stream, Search, and SearchDomain followers do not (D4).
	if follower.ParentType != model.FollowerTypeUser {
		return false
	}

	if follower.Method != model.FollowerMethodEmail {
		return false
	}

	// Nothing can be pushed without an address to push
	return follower.Actor.EmailAddress != ""
}
