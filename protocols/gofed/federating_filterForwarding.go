package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/pub"
)

/*
Allows the implementation to apply outbound message business logic such as blocks, spam filtering, and so on to a list of potentialRecipients when inbox forwarding has been triggered. Your application must apply some sort of filtering, such as limiting delivery to an actor's followers. Otherwise, your application will become a vector for spam on behalf of malicious peers, and users of your software will be mass-blocked by their peers.

The activity is provided as a reference for more intelligent logic to be used, but the implementation must not modify the activity.
*/
func (fed Federating) FilterForwarding(c context.Context, potentialRecipients []*url.URL, a pub.Activity) (filteredRecipients []*url.URL, err error) {
	return nil, nil
}
