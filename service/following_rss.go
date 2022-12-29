package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/mmcdole/gofeed"
)

/*******************************************
 * Connection Methods
 *******************************************/

func (service *Following) import_RSS(following *model.Following, response *http.Response, body *bytes.Buffer) error {

	const location = "service.Following.importRSS"

	// Try to find the RSS feed associated with this link
	rssFeed, err := gofeed.NewParser().ParseString(body.String())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing RSS feed", body.String())
	}

	// Update the label for this "following" record using the RSS feed title.
	// This should get saved once we successfully update the record status.
	following.Label = rssFeed.Title
	following.Links = discoverLinks_RSS(response, body)

	// If we have a feed, then import all of the items from it.

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, rssItem := range rssFeed.Items {
		activity := convert.RSSToActivity(rssFeed, rssItem)
		if err := service.saveActivity(following, &activity); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error updating local activity"))
		}
	}

	// If there were errors parsing the feed, then mark the record as an error.
	if errorCollection != nil {

		// Try to update the following status
		if err := service.SetStatus(following, model.FollowingStatusFailure, errorCollection.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating following status", following)
		}

		// There were errors, but they're noted in the following status, so THIS step is successful
		return nil
	}

	// If we're here, then we have successfully imported the RSS feed.
	// Mark the following as having been polled
	if err := service.SetStatus(following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	return nil
}
