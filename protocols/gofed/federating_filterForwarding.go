package gofed

import (
	"context"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/pub"
)

// Allows the implementation to apply outbound message business logic such as blocks, spam filtering,
// and so on to a list of potentialRecipients when inbox forwarding has been triggered. Your
// application must apply some sort of filtering, such as limiting delivery to an actor's followers.
// Otherwise, your application will become a vector for spam on behalf of malicious peers, and users
// of your software will be mass-blocked by their peers.
//
// The activity is provided as a reference for more intelligent logic to be used, but the
// implementation must not modify the activity.
func (fed Federating) FilterForwarding(c context.Context, potentialRecipients []*url.URL, a pub.Activity) (filteredRecipients []*url.URL, err error) {

	// TODO: HIGH: Implement this after ActivityPub is working, once BLOCKS are in place.
	// For now, NO forwarding is allowed, but this will probably change.
	// Research:
	// https://www.google.com/search?q=ActivityPub+inbox+forwarding
	// https://www.w3.org/TR/activitypub/#inbox-forwarding
	// https://socialhub.activitypub.rocks/t/inbox-forwarding/2580

	spew.Dump("FilterForwarding", potentialRecipients, a)
	return make([]*url.URL, 0), nil
}
