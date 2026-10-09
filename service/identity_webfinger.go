package service

import (
	"crypto/sha256"

	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// sendGuestCode_ActivityPub delivers a guest sign-in code to an actor as an ActivityPub direct message
func (service *Identity) sendGuestCode_ActivityPub(session data.Session, identifier string, recipientID string, code string) error {

	// Create the outbound message
	hostname := service.hostname()

	idHash := sha256.Sum256([]byte(code))
	objectID := service.host + "/@guest/signin/" + string(idHash[:])

	url := service.host + "/@guest/signin/" + code
	publishedDate := datetime.Now()

	content := "Hello " + identifier +
		"<br><br>" +
		"Here is your guest code to sign in to " + hostname + ". " +
		"This code is valid for ONE HOUR." +
		"<br><br>" +
		"To continue, click the link below and you'll be linked back to your guest profile on " + hostname +
		"<br><br>" +
		"<a href='" + url + "' target='_blank' rel='noopener noreferrer'>Click here to Sign In &rarr;</a>"

	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        objectID,
		vocab.PropertyType:      vocab.ActivityTypeCreate,
		vocab.PropertyActor:     service.host + "/@application",
		vocab.PropertyPublished: publishedDate,
		vocab.PropertyTo:        []string{recipientID},
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyType:         vocab.ObjectTypeNote,
			vocab.PropertyID:           objectID,
			vocab.PropertyPublished:    publishedDate,
			vocab.PropertyAttributedTo: service.host + "/@application",
			vocab.PropertyTo:           []string{recipientID},
			vocab.PropertyContent:      content,
			vocab.PropertyTag: []mapof.Any{
				{
					vocab.PropertyType: vocab.LinkTypeMention,
					vocab.PropertyName: identifier,
					vocab.PropertyHref: recipientID,
				},
			},
		},
	}

	// Deliver as a post-commit ActivityPub send, signed as @application. The caller resolved the
	// recipient synchronously before minting the code, so a bad address is still reported in real
	// time; only the signed HTTP POST is deferred to the queue (POST-COMMIT-FEDERATION.md F5).
	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, activity)

	return nil
}
