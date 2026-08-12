package service

import (
	"crypto/sha256"

	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// sendGuestCode_ActivityPub delivers a guest sign-in code to an Identity as an ActivityPub direct message
func (service *Identity) sendGuestCode_ActivityPub(session data.Session, identifier string, code string) error {

	const location = "service.Identity.sendGuestCode_ActivityPub"

	// Find Recipient

	recipientID, _, err := service.activityService.GetRecipient(identifier)

	if err != nil {
		return derp.Wrap(err, location, "Finding recipient inbox", identifier)
	}

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
		"<a href=" + url + " target=_blank>Click here to Sign In &rarr;</a>"

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

	// Deliver the guest code as a post-commit ActivityPub send. GetRecipient above already
	// validated the identifier and resolved its inbox SYNCHRONOUSLY, so a bad/unreachable
	// address is still reported to the caller in real time (driving the "double-check your
	// address" UX); only the signed HTTP POST is deferred to the queue — where a transient
	// failure is now retried instead of lost. The activity is addressed to:[recipient] and
	// signed as @application (SendLocator.Actor resolves the Application actor). This also
	// removes the last signed HTTP send from inside the request transaction, and was the final
	// caller of ActivityStream.SendMessage. See POST-COMMIT-FEDERATION.md F5.
	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, activity)

	return nil
}
