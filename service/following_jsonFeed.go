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
	following.SetLinks(discoverLinks_JSONFeed(response, &feed)...)

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	for _, item := range feed.Items {
		message := convert.JsonFeedToActivity(feed, item)
		if err := service.saveToInbox(following, &message); err != nil {
			return derp.Wrap(err, location, "Error saving message", following, message)
		}
	}

	// Save our success!
	return nil
}
