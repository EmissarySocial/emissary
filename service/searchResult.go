package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
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

// List returns an iterator containing all of the SearchResults that match the provided criteria
func (service *SearchResult) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(criteria, options...)
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

	// Update tags
	if _, tagValues, err := service.searchTagService.NormalizeTags(searchResult.Tags...); err == nil {
		searchResult.Tags = tagValues
	} else {
		return derp.Wrap(err, "service.Search.Save", "Error normalizing tags", searchResult)
	}

	// Make Text Index
	searchResult.Index = TextIndex(searchResult.Text)

	// Reindex this Search in 30 days
	searchResult.ReIndexDate = time.Now().Add(time.Hour * 24 * 30).Unix()

	// Save the searchResult to the database
	if err := service.collection.Save(searchResult, note); err != nil {
		return derp.Wrap(err, "service.Search.Save", "Error saving Search", searchResult, note)
	}

	for _, tagName := range searchResult.Tags {
		if err := service.searchTagService.Upsert(tagName); err != nil {
			return derp.Wrap(err, "service.Search.Save", "Error saving SearchTag", searchResult, tagName)
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

func (service *SearchResult) LoadByURL(url string, searchResult *model.SearchResult) error {
	return service.Load(exp.Equal("url", url), searchResult)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *SearchResult) Sync(searchResult model.SearchResult) error {

	// If the SearchResult is marked as deleted, then remove it from the database
	if searchResult.IsDeleted() {
		return service.DeleteByURL(searchResult.URL)
	}

	// Try to load the original SearchResult
	original := model.NewSearchResult()

	err := service.LoadByURL(searchResult.URL, &original)
	var comment string

	switch {

	// If the SearchResult exists in the database, then update it
	case err == nil:
		original.Update(searchResult)
		comment = "updated"

	// If the SearchResult is NOT FOUND, then insert it.
	case derp.NotFound(err):
		original = searchResult
		comment = "added"

	// Return legitimate errors to the caller
	default:
		return derp.Wrap(err, "service.Search.Upsert", "Error loading Search", searchResult)
	}

	// Save the new/updated SearchResult to the database
	if err := service.Save(&original, comment); err != nil {
		return derp.Wrap(err, "service.Search.Add", "Error adding Search", searchResult)
	}

	// Great Success
	return nil
}

// eleteByURL removes a SearchResult from the database that matches the provided URL
func (service *SearchResult) DeleteByURL(url string) error {

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

		return derp.Wrap(err, "service.Search.DeleteByURL", "Error loading Search", url)
	}

	// Delete the SearchResult
	return service.Delete(&searchResult, "deleted from search index")
}

// Shuffle updates the "shuffle" field for all SearchResults that match the provided tags
func (service *SearchResult) Shuffle(tags ...string) error {

	const location = "service.Search.Shuffle"
	return derp.NewInternalError(location, "Not implemented", tags)

	/*
		rangeFunc, err := service.RangeByTags(tags...)

		if err != nil {
			return derp.Wrap(err, location, "Error listing SearchResults", tags)
		}

		for result := range rangeFunc {
			result.Shuffle = rand.Int63()
			if err := service.Save(&result, "shuffled"); err != nil {
				return derp.Wrap(err, location, "Error saving SearchResult", result)
			}
		}

		return nil
	*/
}
