package service

import (
	"context"
	"iter"
	"math/rand"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult defines a service that manages all searchable pages in a domain.
type SearchResult struct {
	collection       data.Collection
	searchTagService *SearchTag
	host             string
}

// NewSearchResult returns a fully initialized Search service
func NewSearchResult() SearchResult {
	return SearchResult{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchResult) Refresh(collection data.Collection, searchTagService *SearchTag, host string) {
	service.collection = collection
	service.searchTagService = searchTagService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchResult) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchResult) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(criteria)
}

// Query returns an slice of allthe SearchResults that match the provided criteria
func (service *SearchResult) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchResult, error) {
	result := make([]model.SearchResult, 0)
	err := service.collection.Query(&result, criteria, options...)

	return result, err
}

// QueryIDsOnly returns an slice of allthe SearchResults that match the provided criteria
func (service *SearchResult) QueryIDsOnly(criteria exp.Expression, options ...option.Option) ([]model.IDOnly, error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection.Query(&result, criteria, options...)

	return result, err
}

// Range returns a Go RangeFunc that iterates over the SearchResults that match the provided criteria
func (service *SearchResult) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchResult], error) {
	it, err := service.collection.Iterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Search.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchResult), nil
}

// Load retrieves an SearchResult from the database
func (service *SearchResult) Load(criteria exp.Expression, searchResult *model.SearchResult) error {

	if err := service.collection.Load(criteria, searchResult); err != nil {
		return derp.Wrap(err, "service.Search.Load", "Error loading Search", criteria)
	}

	return nil
}

// Save adds/updates an SearchResult in the database
func (service *SearchResult) Save(searchResult *model.SearchResult, note string) error {

	const location = "service.Search.Save"

	if searchResult.SearchResultID.IsZero() {
		return derp.NewInternalError(location, "SearchResultID is required", searchResult)
	}

	// Normalize Tags
	if _, tagValues, err := service.searchTagService.NormalizeTags(searchResult.Tags...); err == nil {
		searchResult.Tags = tagValues
		slices.Sort(searchResult.Tags)
	} else {
		return derp.Wrap(err, location, "Error normalizing tags", searchResult)
	}

	// Make Text Index
	searchResult.Index = TextIndex(searchResult.Text)
	slices.Sort(searchResult.Index)

	// Reindex this Search in 30 days
	searchResult.ReIndexDate = time.Now().Add(time.Hour * 24 * 30).Unix()

	// Save the searchResult to the database
	if err := service.collection.Save(searchResult, note); err != nil {
		return derp.Wrap(err, location, "Error saving Search", searchResult, note)
	}

	for _, tagName := range searchResult.Tags {
		if err := service.searchTagService.Upsert(tagName); err != nil {
			return derp.Wrap(err, location, "Error saving SearchTag", searchResult, tagName)
		}
	}

	return nil
}

// Delete removes an Search from the database (HARD DELETE)
func (service *SearchResult) Delete(searchResult *model.SearchResult, note string) error {

	// Use HARD DELETE for search results.  No need to clutter up our indexes with "deleted" data.
	criteria := exp.Equal("_id", searchResult.SearchResultID)
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Search.Delete", "Error deleting Search", searchResult, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeAll returns an iterator function that loops over ALL SearchResults in the database.
func (service *SearchResult) RangeAll() (iter.Seq[model.SearchResult], error) {
	return service.Range(exp.All())
}

// LoadByURL returns a single SearchResult that matches the provided URL

func (service *SearchResult) LoadByURL(url string, searchResult *model.SearchResult) error {
	return service.Load(exp.Equal("url", url), searchResult)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *SearchResult) Sync(searchResult model.SearchResult) error {

	const location = "service.Search.Sync"

	// If the SearchResult is marked as deleted, then remove it from the database
	if searchResult.IsDeleted() {
		return service.DeleteByURL(searchResult.URL)
	}

	// Try to load the original SearchResult
	original := model.NewSearchResult()
	err := service.LoadByURL(searchResult.URL, &original)

	// If the SearchResult exists in the database, then update it
	if err == nil {
		original.Update(searchResult)
		if err := service.Save(&original, "updated"); err != nil {
			return derp.Wrap(err, location, "Error adding Search", searchResult)
		}

		return nil
	}

	// If the SearchResult is NOT FOUND, then insert it.
	if derp.NotFound(err) {
		if err := service.Save(&searchResult, "added"); err != nil {
			return derp.Wrap(err, location, "Error adding Search", searchResult)
		}

		return nil
	}

	// Return legitimate errors to the caller
	return derp.Wrap(err, location, "Error loading Search", searchResult)
}

// eleteByURL removes a SearchResult from the database that matches the provided URL
func (service *SearchResult) DeleteByURL(url string) error {

	const location = "service.Search.DeleteByURL"

	// RULE: If the URL is empty, then there's nothing to delete
	if url == "" {
		return nil
	}

	// Try to find the SearchResult that matches this URL
	searchResult := model.NewSearchResult()

	if err := service.LoadByURL(url, &searchResult); err != nil {

		if derp.NotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error loading Search", url)
	}

	// Delete the SearchResult
	return service.Delete(&searchResult, "deleted from search index")
}

// Shuffle updates the "shuffle" field for all SearchResults that match the provided tags
func (service *SearchResult) Shuffle() error {

	const location = "service.Search.Shuffle"

	rangeFunc, err := service.RangeAll()

	if err != nil {
		return derp.Wrap(err, location, "Error listing SearchResults")
	}

	for result := range rangeFunc {
		result.Shuffle = rand.Int63()
		if err := service.Save(&result, "shuffled"); err != nil {
			return derp.Wrap(err, location, "Error saving SearchResult", result)
		}
	}

	return nil
}

// GetResultsToNotify locks a batch of SearchResults and returns it to the caller.
func (service *SearchResult) GetResultsToNotify(lockID primitive.ObjectID) ([]model.SearchResult, error) {

	const location = "service.Search.GetLockedResults"

	// Make a timeout context for this request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find a batch of UNLOCKED search results
	searchResultIDs, err := service.QueryUnnotifiedAndUnlocked()

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading search results to lock")
	}

	// If there are no matching results, then exit early
	if len(searchResultIDs) == 0 {
		return make([]model.SearchResult, 0), nil
	}

	// Try to lock a batch of search results (up to 32, maybe less)
	if err := queries.LockSearchResults(ctx, service.collection, searchResultIDs, lockID); err != nil {
		return nil, derp.Wrap(err, location, "Error locking search results", searchResultIDs)
	}

	// Load all of the search results that are locked by this process (up to 32, maybe less)
	criteria := exp.Equal("lockId", lockID)
	return service.Query(criteria)
}

// QueryUnnotifiedandUnlocked returns the IDs of the first 32 SearchResults that have NOT been notified, and are NOT locked.
func (service *SearchResult) QueryUnnotifiedAndUnlocked() ([]primitive.ObjectID, error) {

	const location = "service.Search.QueryUnnotifiedAndUnlocked"

	result, err := service.QueryIDsOnly(
		exp.Equal("notifiedDate", 0).
			AndLessThan("timeoutDate", time.Now().Unix()),
		option.MaxRows(32),
		option.SortAsc("createDate"),
	)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading search results to lock")
	}

	return model.GetIDOnly(result), nil
}
