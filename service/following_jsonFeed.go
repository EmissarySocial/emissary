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

	var feed jsonfeed.Feed

	if err := json.Unmarshal(body.Bytes(), &feed); err != nil {
		return derp.Wrap(err, "service.Following.importJSONFeed", "Error parsing JSON Feed", following, body.String())
	}

	following.Label = feed.Title

	for _, item := range feed.Items {
		stream := convert.JsonFeedToStream(item)

		if err := service.saveStream(following, &stream); err != nil {
			return derp.Wrap(err, "service.Following.importJSONFeed", "Error saving stream", following, stream)
		}
	}

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

		// Unrecognized hub type
		default:
			continue
		}

		// If we're here, we found a hub that we can use, so we're done.
		break
	}

	// Save our success
	if err := service.SetStatus(following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, "service.Following.importJSONFeed", "Error setting status", following)
	}

	return nil
}
