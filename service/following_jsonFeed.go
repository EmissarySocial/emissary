package service

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/kr/jsonfeed"
)

func (service *Following) import_JSONFeed(following *model.Following, response *http.Response, body *bytes.Buffer) error {

	const location = "service.Following.importJSONFeed"

	var feed jsonfeed.Feed

	// Parse the JSON feed
	if err := json.Unmarshal(body.Bytes(), &feed); err != nil {
		return derp.Wrap(err, location, "Error parsing JSON Feed", following, body.String())
	}

	following.Label = feed.Title
	following.Links = discoverLinks_JSONFeed(response, &feed)

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, item := range feed.Items {
		activity := convert.JsonFeedToActivity(item)
		if err := service.saveActivity(following, &activity); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error saving activity", following, activity))
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

	// Save our success
	if err := service.SetStatus(following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error setting status", following)
	}

	return nil
}
