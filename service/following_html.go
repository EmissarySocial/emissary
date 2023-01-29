package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/davecgh/go-spew/spew"
	"willnorris.com/go/microformats"
)

func (service *Following) import_HTML(following *model.Following, response *http.Response, body *bytes.Buffer) error {

	const location = "service.Following.importHTML"

	// Look for Feed Data
	if err := service.import_HTML_feed(following, response, body); err != nil {
		return derp.Wrap(err, location, "Error importing HTML", following, body.String())
	}

	// Success!
	return nil
}

func (service *Following) import_HTML_feed(following *model.Following, response *http.Response, body *bytes.Buffer) error {

	for _, mediaType := range []string{model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText} {

		if link := following.Links.FindBy("type", mediaType); !link.IsEmpty() {

			spew.Dump(mediaType)

			switch link.MediaType {

			case model.MimeTypeJSONFeed:
				if err := service.poll(following, link, service.import_JSONFeed); err == nil {
					return nil
				} else {
					derp.Report(err)
				}

			case model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText:
				if err := service.poll(following, link, service.import_RSS); err == nil {
					return nil
				} else {
					derp.Report(err)
				}
			}
		}
	}

	// Last ditch: Scan the body for a microformat h-feed
	if service.import_Microformats(following, response, body) {
		return nil
	}

	return derp.NewBadRequestError("service.following.import_HTML_feed", "No feed or links found in HTML document", following)
}

func (service *Following) import_Microformats(following *model.Following, response *http.Response, body *bytes.Buffer) bool {

	var atLeastOneChild bool
	data := microformats.Parse(bytes.NewReader(body.Bytes()), response.Request.URL)

	for _, feed := range data.Items {

		if slice.Contains(feed.Type, "h-feed") {
			following.Label = convert.MicroformatPropertyToString(feed, "name")

			for _, child := range feed.Children {
				if slice.Contains(child.Type, "h-entry") {
					atLeastOneChild = true
					activity := convert.MicroformatToActivity(feed, child)
					if err := service.saveActivity(following, &activity); err != nil {
						derp.Report(err) // report, but swallow error details
					}
				}
			}
		}
	}

	return atLeastOneChild
}
