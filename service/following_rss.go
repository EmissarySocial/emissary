package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/mmcdole/gofeed"
)

/*******************************************
 * Connection Methods
 *******************************************/

// PollRSS tries to import an RSS feed and adds/updates activitys for each item in it.
func (service *Following) PollRSS(following *model.Following, link digit.Link) error {

	const location = "service.Following.PollRSS"

	// If this is not an RSS following, then just shut that whole thing down...
	if following.Method != model.FollowMethodRSS {
		return nil
	}

	// Try to find the RSS feed associated with this link
	rssFeed, err := gofeed.NewParser().ParseURL(link.Href)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing RSS feed", link.Href)
	}

	// Update the label for this "following" record using the RSS feed title.
	// This should get saved once we successfully update the record status.
	following.Label = rssFeed.Title

	// If we have a feed, then import all of the items from it.

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, item := range rssFeed.Items {
		if err := service.saveActivity(following, rssFeed, item); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error updating local activity"))
		}
	}

	// If there were errors parsing the feed, then mark the following as an error.
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

// saveActivity adds/updates an individual Activity based on an RSS item
func (service *Following) saveActivity(following *model.Following, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	const location = "service.Following.saveActivity"

	activity := model.NewActivity()

	// Look for duplicate records.  404 error means "no duplicate" so we can create a new one.
	if err := service.inboxService.LoadByDocumentURL(following.UserID, rssItem.Link, &activity); err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading local activity")
		}

		// Fall through means "not found" which means "make a new activity"
		activity.OwnerID = following.UserID
		activity.Origin = following.Origin()
		activity.PublishDate = rssDate(rssItem.PublishedParsed)
		activity.FolderID = following.FolderID

		if updateDate := rssDate(rssItem.UpdatedParsed); updateDate > activity.PublishDate {
			activity.PublishDate = updateDate
		}
	}

	// If the RSS entry has been updated since the Activity was last touched, then refresh it.
	if rssDate(rssItem.PublishedParsed) >= activity.Journal.UpdateDate {

		populateActivity(&activity, following, rssFeed, rssItem)

		// Try to save the new/updated activity
		if err := service.inboxService.Save(&activity, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "service.Following.Poll", "Error saving activity")
		}
	}

	return nil
}
