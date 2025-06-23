package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

func (service *Identity) sendGuestCode_Webfinger(identifier string, code string) error {

	const location = "service.Identity.sendGuestCode_Webfinger"

	// Find Recipient
	recipientID, inboxURL, err := service.activityService.GetRecipient(identifier)

	if err != nil {
		return derp.Wrap(err, location, "Error finding recipient inbox", inboxURL)
	}

	// Create the outbound message
	hostname := service.hostname()

	url := service.host + "/@guest/signin/" + code
	publishedDate := hannibal.TimeFormat(time.Now())

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
		vocab.PropertyType:      vocab.ActivityTypeCreate,
		vocab.PropertyActor:     service.host + "/@application",
		vocab.PropertyPublished: publishedDate,
		vocab.PropertyTo:        []string{recipientID},
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyType:      vocab.ObjectTypeNote,
			vocab.PropertyPublished: publishedDate,
			vocab.PropertyTo:        []string{recipientID},
			vocab.PropertyContent:   content,
			vocab.PropertyTag: []mapof.Any{
				{
					vocab.PropertyType: vocab.LinkTypeMention,
					vocab.PropertyName: identifier,
					vocab.PropertyHref: recipientID,
				},
			},
		},
	}

	message := mapof.Any{
		"host":      hostname,
		"actorType": model.FollowerTypeApplication,
		"inboxURL":  inboxURL,
		"message":   activity,
	}

	// Because we want a real-time response, we're going to run this queue task inline
	if err := service.activityService.SendMessage(message); err != nil {
		return derp.Wrap(err, location, "Error sending guest code to WebFinger identifier", identifier)
	}

	return nil
}
