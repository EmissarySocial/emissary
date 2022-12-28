package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/kr/jsonfeed"
)

func (service *Following) import_JSONFeed(following *model.Following, _ *http.Response, body *bytes.Buffer) error {

	const location = "service.Following.importJSONFeed"

	var feed jsonfeed.Feed

	// Parse the JSON feed
	if err := json.Unmarshal(body.Bytes(), &feed); err != nil {
		return derp.Wrap(err, location, "Error parsing JSON Feed", following, body.String())
	}

	following.Label = feed.Title

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, item := range feed.Items {
		activity := convert.JsonFeedToActivity(item)
		if err := service.saveActivity(following, &activity); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error saving activity", following, activity))
		}
	}

	if errorCollection != nil {

		// Try to update the following status
		if err := service.SetStatus(following, model.FollowingStatusFailure, errorCollection.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating following status", following)
		}

		// There were errors, but they're noted in the following status, so THIS step is successful
		return nil
	}

	// Discover hubs
	for _, hub := range feed.Hubs {

		switch strings.ToUpper(hub.Type) {

		case model.FollowMethodWebSub:
			link := digit.NewLink(model.LinkRelationHub, model.MimeTypeJSONFeed, hub.URL)
			if err := service.connect_WebSub(following, link, following.URL); err != nil {
				continue
			}

		case model.FollowMethodRSSCloud:
			link := digit.NewLink(model.LinkRelationHub, model.MimeTypeJSONFeed, hub.URL)
			if err := service.connect_RSSCloud(following, link); err != nil {
				continue
			}

		default:
			continue
		}

		// If we're here, we found a hub that we can use, so we're done.
		break
	}

	// Save our success
	if err := service.SetStatus(following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error setting status", following)
	}

	return nil
}
