package service

/******************************************
 * Outbox2 Sends
 *
 * Activities that a local actor sends without storing a row:
 * answers to follow requests, follows and unfollows, and a
 * Stream's rebroadcasts. Each one is built here and handed to
 * Send, which queues hannibal's fan-out after the commit.
 ******************************************/

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// SendAccept sends an "Accept" of an activity, usually a Follow, to that activity's actor.
// actorURL is the accepting local actor.
func (service *Outbox2) SendAccept(session data.Session, actorURL string, acceptID string, activity streams.Document) error {

	const location = "service.Outbox2.SendAccept"

	// RULE: Read the actors' ids where they are written. Actor() loads each one, which needs a client
	// and a fetch, and a Document with no client would address the Accept to no one.
	accept := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyID:     acceptID,
		vocab.PropertyType:   vocab.ActivityTypeAccept,
		vocab.PropertyActor:  actorURL,
		vocab.PropertyObject: activity.Map(),
		vocab.PropertyTo:     activity.Get(vocab.PropertyActor).IDs(),
	}

	if err := service.Send(session, accept); err != nil {
		return derp.Wrap(err, location, "Sending Accept", "acceptId: "+acceptID)
	}

	return nil
}

// SendRejectFollow sends a "Reject" of a follower's Follow to that follower, so that their server
// knows the follow has ended. actorURL is the local actor that was followed.
func (service *Outbox2) SendRejectFollow(session data.Session, actorURL string, rejectID string, followerURL string) error {

	const location = "service.Outbox2.SendRejectFollow"

	// Mastodon matches a Reject on the Follow's actor and object, so the Follow's own id is not needed
	reject := mapof.Any{
		vocab.AtContext:     vocab.ContextTypeActivityStreams,
		vocab.PropertyID:    rejectID,
		vocab.PropertyType:  vocab.ActivityTypeReject,
		vocab.PropertyActor: actorURL,
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyType:   vocab.ActivityTypeFollow,
			vocab.PropertyActor:  followerURL,
			vocab.PropertyObject: actorURL,
		},
		vocab.PropertyTo: []string{followerURL},
	}

	if err := service.Send(session, reject); err != nil {
		return derp.Wrap(err, location, "Sending Reject", "rejectId: "+rejectID)
	}

	return nil
}

// SendFollow sends a "Follow" of remoteActorID to that actor. actorURL is the local actor who follows.
func (service *Outbox2) SendFollow(session data.Session, actorURL string, followID string, remoteActorID string) error {

	const location = "service.Outbox2.SendFollow"

	follow := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        followID,
		vocab.PropertyType:      vocab.ActivityTypeFollow,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    remoteActorID,
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        []string{remoteActorID},
	}

	if err := service.Send(session, follow); err != nil {
		return derp.Wrap(err, location, "Sending Follow", "followId: "+followID)
	}

	return nil
}

// SendUndoFollow sends an "Undo" of a Follow to the actor that was followed. actorURL is the local
// actor who followed, and follow is the original Follow, embedded so the remote server knows which to reverse.
func (service *Outbox2) SendUndoFollow(session data.Session, actorURL string, follow mapof.Any) error {

	const location = "service.Outbox2.SendUndoFollow"

	// The Undo has no id of its own, as before; the fan-out keys its tasks on a hash of the body instead
	undo := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      vocab.ActivityTypeUndo,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    follow,
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        streams.NewDocument(follow).Get(vocab.PropertyObject).IDs(),
	}

	if err := service.Send(session, undo); err != nil {
		return derp.Wrap(err, location, "Sending Undo of a Follow", "followId: "+follow.GetString(vocab.PropertyID))
	}

	return nil
}

// SendAnnounce sends an "Announce" of object as a public boost, to Public and the announcing actor's
// followers. actorURL is the announcing local actor, and followersURL is its followers collection.
func (service *Outbox2) SendAnnounce(session data.Session, actorURL string, announceID string, object streams.Document, followersURL string) error {

	const location = "service.Outbox2.SendAnnounce"

	// A boost reaches the booster's followers and Public, never the original post's private audience
	announce := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        announceID,
		vocab.PropertyType:      vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    object.Map(),
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        []string{vocab.NamespacePublic},
		vocab.PropertyCC:        []string{followersURL},
	}

	if err := service.Send(session, announce); err != nil {
		return derp.Wrap(err, location, "Sending Announce", "announceId: "+announceID)
	}

	return nil
}
