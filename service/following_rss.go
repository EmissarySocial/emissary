package service

/******************************************
 * Connection Methods

func (service *Following) import_RSS(following *model.Following, response *http.Response, body *bytes.Buffer) error {

	const location = "service.Following.importRSS"

	// Try to find the RSS feed associated with this link
	rssFeed, err := gofeed.NewParser().ParseString(body.String())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing RSS feed")
	}

	// Update the label for this "following" record using the RSS feed title.
	// This should get saved once we successfully update the record status.
	following.Label = rssFeed.Title

	if rssFeed.Image != nil {
		if imageURL := rssFeed.Image.URL; imageURL != "" {
			following.ImageURL = imageURL
		}
	}

	if following.ImageURL == "" {
		following.ImageURL = following.Links.FindBy("rel", "icon").Href
	}

	following.SetLinks(discoverLinks_RSS(response, body)...)

	// If we have a feed, then import all of the items from it.

	// Before inserting, sort the items chronologically so that new feeds appear correctly in the UX
	sort.SliceStable(rssFeed.Items, func(i, j int) bool {
		return rssFeed.Items[i].PublishedParsed.Unix() < rssFeed.Items[j].PublishedParsed.Unix()
	})

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	recalculate := false
	for _, rssItem := range rssFeed.Items {
		defaultValue := convert.RSSToActivity(rssFeed, rssItem)
		if document, err := service.httpClient.LoadDocument(rssItem.Link, defaultValue); err == nil {
			if created, err := service.saveDocument(following, &document); err != nil {
				derp.Report(derp.Wrap(err, location, "Error saving document", document))
			} else if created {
				recalculate = true
			}
		} else {
			derp.Report(derp.Wrap(err, location, "Error loading document", rssItem.Link))
		}
	}

	// Recalculate read counts if we've added any new items to the feed.
	if recalculate {
		if err := service.folderService.ReCalculateUnreadCountFromFolder(following.UserID, following.FolderID); err != nil {
			return derp.Wrap(err, location, "Error updating read counts")
		}
	}

	return nil
}

******************************************/
