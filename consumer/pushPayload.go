package consumer

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// pushPayloads holds a Notification encoded once for each kind of push subscription
type pushPayloads struct {
	web    []byte
	mobile []byte
}

// newPushPayloads encodes a Notification for browsers and for mobile apps
func newPushPayloads(host string, notification *model.Notification) (pushPayloads, error) {

	const location = "consumer.newPushPayloads"

	web, err := json.Marshal(newWebPushPayload(host, notification))

	if err != nil {
		return pushPayloads{}, derp.Wrap(err, location, "Marshaling web push payload")
	}

	mobile, err := json.Marshal(newMobilePushPayload(notification)) // #nosec G117 -- Mastodon's own field, always sent empty

	if err != nil {
		return pushPayloads{}, derp.Wrap(err, location, "Marshaling mobile push payload")
	}

	return pushPayloads{web: web, mobile: mobile}, nil
}
