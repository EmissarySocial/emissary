package service

import (
	"context"
	"iter"
	"math"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/EmissarySocial/emissary/tools/sorted"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult defines a service that manages all searchable pages in a domain.
type SearchResult struct {
	searchTagService *SearchTag
	queue            *queue.Queue
	hostname         string
}

// NewSearchResult returns a fully initialized Search service
func NewSearchResult() SearchResult {
	return SearchResult{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchResult) Refresh(searchTagService *SearchTag, queue *queue.Queue, hostname string) {
	service.searchTagService = searchTagService
	service.queue = queue
	service.hostname = hostname
}

// Close stops any background processes controlled by this service
func (service *SearchResult) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchResult) collection(session data.Session) data.Collection {
	return session.Collection("SearchResult")
}

func (service *SearchResult) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(criteria)
}

// Query returns an slice of allthe SearchResults that match the provided criteria
func (service *SearchResult) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.SearchResult, error) {
	result := make([]model.SearchResult, 0)
	err := service.collection(session).Query(&result, criteria, options...)

	return result, err
}

// QueryIDsOnly returns an slice of allthe SearchResults that match the provided criteria
func (service *SearchResult) QueryIDsOnly(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.IDOnly, error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, criteria, options...)

	return result, err
}

// Range returns a Go RangeFunc that iterates over the SearchResults that match the provided criteria
func (service *SearchResult) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchResult], error) {
	it, err := service.collection(session).Iterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Search.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchResult), nil
}

// Load retrieves an SearchResult from the database
func (service *SearchResult) Load(session data.Session, criteria exp.Expression, searchResult *model.SearchResult) error {

	if err := service.collection(session).Load(criteria, searchResult); err != nil {
		return derp.Wrap(err, "service.Search.Load", "Error loading Search", criteria)
	}

	return nil
}

// Save adds/updates an SearchResult in the database
func (service *SearchResult) Save(session data.Session, searchResult *model.SearchResult, note string) error {

	const location = "service.Search.Save"

	// RULE: Do not save empty SearchResults
	if searchResult.SearchResultID.IsZero() {
		return derp.InternalError(location, "SearchResultID is required", searchResult)
	}

	// RULE: If unassigned, shuffle the searchResult after the first trillion other results (will reset in 1 hour)
	if searchResult.Shuffle == 0 {
		searchResult.Shuffle = math.MaxInt64 - int64(random.GenerateInt(0, 999_999_999_999))
	}

	// Normalize Tags
	_, tagValues, err := service.searchTagService.NormalizeTags(session, searchResult.Tags...)

	if err != nil {
		return derp.Wrap(err, location, "Error normalizing tags", searchResult)
	}

	wasNew := searchResult.IsNew()

	// Make Tags Index
	slices.Sort(tagValues)
	searchResult.Tags = sorted.Unique(tagValues)

	textTokens := parse.Split(searchResult.Text)
	textTokens = append(textTokens, searchResult.Tags...)

	// Make Text Index (which includes tags)
	textIndex := textIndex(textTokens...)
	slices.Sort(textIndex)
	searchResult.Index = sorted.Unique(textIndex)

	// Reindex this Search in 30 days
	searchResult.ReIndexDate = time.Now().Add(time.Hour * 24 * 30).Unix()

	// Save the searchResult to the database
	if err := service.collection(session).Save(searchResult, note); err != nil {
		return derp.Wrap(err, location, "Error saving Search", searchResult, note)
	}

	for _, tagName := range searchResult.Tags {
		if err := service.searchTagService.Upsert(session, tagName); err != nil {
			return derp.Wrap(err, location, "Error saving SearchTag", searchResult, tagName)
		}
	}

	service.queue.NewTask(
		"SendSearchResult",
		mapof.Any{
			"host":           service.hostname,
			"activity":       iif(wasNew, vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate),
			"searchResultId": searchResult.SearchResultID,
		},
	)

	return nil
}

// Delete removes an Search from the database (HARD DELETE)
func (service *SearchResult) Delete(session data.Session, searchResult *model.SearchResult, note string) error {

	// Use HARD DELETE for search results.  No need to clutter up our indexes with "deleted" data.
	criteria := exp.Equal("_id", searchResult.SearchResultID)
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Search.Delete", "Error deleting Search", searchResult, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID returns a single SearchResult that matches the provided searchResultID
func (service *SearchResult) LoadByID(session data.Session, searchResultID primitive.ObjectID, searchResult *model.SearchResult) error {
	return service.Load(session, exp.Equal("_id", searchResultID), searchResult)
}

// LoadByURL returns a single SearchResult that matches the provided URL
func (service *SearchResult) LoadByURL(session data.Session, url string, searchResult *model.SearchResult) error {
	return service.Load(session, exp.Equal("url", url), searchResult)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *SearchResult) Sync(session data.Session, searchResult model.SearchResult) error {

	const location = "service.Search.Sync"

	// If the SearchResult is marked as deleted, then remove it from the database
	if searchResult.IsDeleted() {

		if err := service.DeleteByURL(session, searchResult.URL); err != nil {
			return derp.Wrap(err, location, "Error deleting Search", searchResult)
		}

		return nil
	}

	// Try to load the original SearchResult
	original := model.NewSearchResult()
	err := service.LoadByURL(session, searchResult.URL, &original)

	// If the SearchResult exists in the database, then update it
	if err == nil {

		// If the original SearchResult has been updated, then also reset the NotifiedDate.
		changed := original.Update(searchResult)

		if changed {
			original.NotifiedDate = 0
		}

		// Save the updated SearchResult...
		if err := service.Save(session, &original, "updated"); err != nil {
			return derp.Wrap(err, location, "Error adding Search", searchResult)
		}

		return nil
	}

	// If the SearchResult is NOT FOUND, then insert it.
	if derp.IsNotFound(err) {
		if err := service.Save(session, &searchResult, "added"); err != nil {
			return derp.Wrap(err, location, "Error adding Search", searchResult)
		}

		return nil
	}

	// Return legitimate errors to the caller
	return derp.Wrap(err, location, "Error loading Search", searchResult)
}

// DeleteByURL removes a SearchResult from the database that matches the provided URL
func (service *SearchResult) DeleteByURL(session data.Session, url string) error {

	const location = "service.Search.DeleteByURL"

	// RULE: If the URL is empty, then there's nothing to delete
	if url == "" {
		return nil
	}

	// Try to find the SearchResult that matches this URL
	searchResult := model.NewSearchResult()

	if err := service.LoadByURL(session, url, &searchResult); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error loading Search", url)
	}

	// Delete the SearchResult
	return service.Delete(session, &searchResult, "deleted from search index")
}

// Shuffle updates the "shuffle" field for all SearchResults that match the provided tags
func (service *SearchResult) Shuffle(session data.Session) error {

	const location = "service.Search.Shuffle"

	collection := service.collection(session)

	if err := queries.Shuffle(session.Context(), collection); err != nil {
		return derp.Wrap(err, location, "Error shuffling SearchResults")
	}

	return nil
}

// GetResultsToNotify locks a batch of SearchResults and returns it to the caller.
func (service *SearchResult) GetResultsToNotify(session data.Session, lockID primitive.ObjectID) ([]model.SearchResult, error) {

	const location = "service.Search.GetLockedResults"

	// Make a timeout context for this request
	ctx, cancel := context.WithTimeout(session.Context(), 30*time.Second)
	defer cancel()

	// Find a batch of UNLOCKED search results
	searchResultIDs, err := service.QueryUnnotifiedAndUnlocked(session)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading search results to lock")
	}

	// If there are no matching results, then exit early
	if len(searchResultIDs) == 0 {
		return make([]model.SearchResult, 0), nil
	}

	// Try to lock a batch of search results (up to 32, maybe less)
	collection := service.collection(session)
	if err := queries.LockSearchResults(ctx, collection, searchResultIDs, lockID); err != nil {
		return nil, derp.Wrap(err, location, "Error locking search results", searchResultIDs)
	}

	// Load all of the search results that are locked by this process (up to 32, maybe less)
	criteria := exp.Equal("lockId", lockID)
	return service.Query(session, criteria)
}

// QueryUnnotifiedandUnlocked returns the IDs of the first 32 SearchResults that have NOT been notified, and are NOT locked.
func (service *SearchResult) QueryUnnotifiedAndUnlocked(session data.Session) ([]primitive.ObjectID, error) {

	const location = "service.Search.QueryUnnotifiedAndUnlocked"

	result, err := service.QueryIDsOnly(
		session,
		exp.Equal("notifiedDate", 0).AndLessThan("timeoutDate", time.Now().Unix()),
		option.MaxRows(32),
	)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading search results to lock")
	}

	return model.GetIDOnly(result), nil
}
