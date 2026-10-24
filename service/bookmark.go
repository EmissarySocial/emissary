package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Bookmark manages the posts that each local User has saved for later.
type Bookmark struct{}

// NewBookmark returns a fully initialized Bookmark service
func NewBookmark() Bookmark {
	return Bookmark{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Bookmark) Refresh(factory *Factory) {
	// Nothing to refresh.
}

// Close stops any background processes controlled by this service
func (service *Bookmark) Close() {
	// Nothing to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Bookmark collection for the provided database session
func (service *Bookmark) collection(session data.Session) data.Collection {
	return session.Collection("Bookmark")
}

// Query returns a slice of Bookmarks that match the provided criteria
func (service *Bookmark) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Bookmark, error) {
	result := make([]model.Bookmark, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Load retrieves a Bookmark from the database
func (service *Bookmark) Load(session data.Session, criteria exp.Expression, bookmark *model.Bookmark) error {

	if err := service.collection(session).Load(notDeleted(criteria), bookmark); err != nil {
		return derp.Wrap(err, "service.Bookmark.Load", "Loading Bookmark", criteria)
	}

	return nil
}

// Save adds/updates a Bookmark in the database
func (service *Bookmark) Save(session data.Session, bookmark *model.Bookmark, note string) error {

	const location = "service.Bookmark.Save"

	if _, err := service.Schema().Validate(bookmark); err != nil {
		return derp.Wrap(err, location, "Validating Bookmark", bookmark)
	}

	if err := service.collection(session).Save(bookmark, note); err != nil {
		return derp.Wrap(err, location, "Saving Bookmark", bookmark, note)
	}

	return nil
}

// Delete removes a Bookmark from the database (hard delete)
func (service *Bookmark) Delete(session data.Session, bookmark *model.Bookmark) error {

	const location = "service.Bookmark.Delete"

	// Hard delete, never virtual: a bookmark is private to its owner, and has no
	// ActivityPub identity that a tombstone would need to keep answering for.
	if err := service.collection(session).HardDelete(exp.Equal("_id", bookmark.BookmarkID)); err != nil {
		return derp.Wrap(err, location, "Deleting Bookmark", bookmark)
	}

	return nil
}

// Schema returns the rosetta schema that describes a Bookmark
func (service *Bookmark) Schema() schema.Schema {
	return schema.New(model.BookmarkSchema())
}

/******************************************
 * Custom Queries + Behaviors
 ******************************************/

// QueryByUser returns the User's bookmarks, newest first.
func (service *Bookmark) QueryByUser(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Bookmark, error) {

	criteria = exp.And(criteria, exp.Equal("userId", userID))
	options = append(options, option.SortDesc("createDate"))

	return service.Query(session, criteria, options...)
}

// LoadByUserAndURL loads the User's bookmark for a single post.
func (service *Bookmark) LoadByUserAndURL(session data.Session, userID primitive.ObjectID, url string, bookmark *model.Bookmark) error {
	return service.Load(session, exp.Equal("userId", userID).AndEqual("url", url), bookmark)
}

// Set bookmarks a post for a User.  It is idempotent: bookmarking an already-bookmarked post is a no-op.
func (service *Bookmark) Set(session data.Session, userID primitive.ObjectID, url string) error {

	const location = "service.Bookmark.Set"

	bookmark := model.NewBookmark()

	err := service.LoadByUserAndURL(session, userID, url, &bookmark)

	if err == nil {
		return nil
	}

	if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Loading Bookmark", url)
	}

	bookmark.UserID = userID
	bookmark.URL = url

	if err := service.Save(session, &bookmark, "Bookmarked via Mastodon API"); err != nil {
		return derp.Wrap(err, location, "Saving Bookmark", url)
	}

	return nil
}

// Unset removes a User's bookmark from a post.  It is idempotent: a post that isn't bookmarked is a no-op.
func (service *Bookmark) Unset(session data.Session, userID primitive.ObjectID, url string) error {

	const location = "service.Bookmark.Unset"

	bookmark := model.NewBookmark()

	if err := service.LoadByUserAndURL(session, userID, url, &bookmark); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading Bookmark", url)
	}

	return service.Delete(session, &bookmark)
}

// FilterBookmarked returns the subset of the provided URLs that the User has bookmarked.
func (service *Bookmark) FilterBookmarked(session data.Session, userID primitive.ObjectID, urls []string) (map[string]bool, error) {

	result := make(map[string]bool, len(urls))

	if len(urls) == 0 {
		return result, nil
	}

	bookmarks, err := service.Query(session, exp.Equal("userId", userID).AndIn("url", urls))

	if err != nil {
		return nil, derp.Wrap(err, "service.Bookmark.FilterBookmarked", "Querying Bookmarks")
	}

	for _, bookmark := range bookmarks {
		result[bookmark.URL] = true
	}

	return result, nil
}
