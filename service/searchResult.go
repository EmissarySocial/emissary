package service

import (
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
		return nil, derp.Wrap(err, "service.SearchResult.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchResult), nil
}

// Load retrieves an SearchResult from the database
func (service *SearchResult) Load(session data.Session, criteria exp.Expression, searchResult *model.SearchResult) error {

	if err := service.collection(session).Load(criteria, searchResult); err != nil {
		return derp.Wrap(err, "service.SearchResult.Load", "Unable to load SearchResult", criteria)
	}

	return nil
}

// Save adds/updates an SearchResult in the database
func (service *SearchResult) Save(session data.Session, searchResult *model.SearchResult, note string) error {

	const location = "service.SearchResult.Save"

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
		return derp.Wrap(err, location, "Unable to save Search", searchResult, note)
	}

	for _, tagName := range searchResult.Tags {
		if err := service.searchTagService.Upsert(session, tagName); err != nil {
			return derp.Wrap(err, location, "Unable to save SearchTag", searchResult, tagName)
		}
	}

	service.queue.NewTask(
		"SendSearchResult",
		mapof.Any{
			"host":           service.hostname,
			"searchResultId": searchResult.SearchResultID,
		},
	)

	return nil
}

// Delete removes an Search from the database (HARD DELETE)
func (service *SearchResult) Delete(session data.Session, searchResult *model.SearchResult, note string) error {

	const location = "service.SearchResult.Delete"

	criteria := exp.Equal("_id", searchResult.SearchResultID)

	// Use HARD DELETE for search results.  No need to clutter up our indexes with "deleted" data.
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete SearchResult", searchResult, note)
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

// Sync matches the provided SearchResult with the URL of a record in the database
// and inserts/updates/deletes the database to match the provided value.
func (service *SearchResult) Sync(session data.Session, searchResult model.SearchResult) error {

	const location = "service.SearchResult.Sync"

	// If the SearchResult is marked as deleted, then remove it from the database
	if searchResult.IsDeleted() {

		if err := service.DeleteByURL(session, searchResult.URL); err != nil {
			return derp.Wrap(err, location, "Unable to delete SearchResult", searchResult)
		}

		return nil
	}

	// Try to load the original SearchResult
	original := model.NewSearchResult()
	err := service.LoadByURL(session, searchResult.URL, &original)

	// If the SearchResult exists in the database, then update it
	if err == nil {

		// If the result is the same as what we already have
		// in the database, then exit here.
		if changed := original.Update(searchResult); !changed {
			return nil
		}

		// Save the updated SearchResult...
		if err := service.Save(session, &original, "updated"); err != nil {
			return derp.Wrap(err, location, "Unable to update SearchResult", searchResult)
		}

		return nil
	}

	// If the SearchResult is NOT FOUND, then insert it.
	if derp.IsNotFound(err) {

		if err := service.Save(session, &searchResult, "added"); err != nil {
			return derp.Wrap(err, location, "Unable to insert SearchResult", searchResult)
		}

		return nil
	}

	// Return legitimate errors to the caller
	return derp.Wrap(err, location, "Unable to query SearchResult", searchResult)
}

// DeleteByURL removes a SearchResult from the database that matches the provided URL
func (service *SearchResult) DeleteByURL(session data.Session, url string) error {

	const location = "service.SearchResult.DeleteByURL"

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

		return derp.Wrap(err, location, "Unable to query SearchResult", url)
	}

	// Delete the SearchResult
	return service.Delete(session, &searchResult, "deleted from search index")
}

// Shuffle updates the "shuffle" field for all SearchResults that match the provided tags
func (service *SearchResult) Shuffle(session data.Session) error {

	const location = "service.SearchResult.Shuffle"

	collection := service.collection(session)

	if err := queries.Shuffle(session.Context(), collection); err != nil {
		return derp.Wrap(err, location, "Error shuffling SearchResults")
	}

	return nil
}
