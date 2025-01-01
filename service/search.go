package service

import (
	"iter"
	"math/rand"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// Search defines a service that manages all searchable pages in a domain.
type Search struct {
	collection       data.Collection
	searchTagService *SearchTag
	host             string
}

// NewSearch returns a fully initialized Search service
func NewSearch() Search {
	return Search{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Search) Refresh(collection data.Collection, searchTagService *SearchTag, host string) {
	service.collection = collection
	service.searchTagService = searchTagService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Search) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Search) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(criteria)
}

// Query returns an slice of allthe Searchs that match the provided criteria
func (service *Search) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchResult, error) {
	result := make([]model.SearchResult, 0)
	err := service.collection.Query(&result, criteria, options...)

	return result, err
}

// List returns an iterator containing all of the Searchs that match the provided criteria
func (service *Search) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(criteria, options...)
}

func (service *Search) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchResult], error) {
	it, err := service.collection.Iterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Search.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchResult), nil
}

// Load retrieves an Search from the database
func (service *Search) Load(criteria exp.Expression, searchResult *model.SearchResult) error {

	if err := service.collection.Load(criteria, searchResult); err != nil {
		return derp.Wrap(err, "service.Search.Load", "Error loading Search", criteria)
	}

	return nil
}

// Save adds/updates an Search in the database
func (service *Search) Save(searchResult *model.SearchResult, note string) error {

	// Reindex this Search in 30 days
	searchResult.ReIndexDate = time.Now().Add(time.Hour * 24 * 30).Unix()

	/*/ Validate the value before saving
	if err := service.Schema().Validate(searchResult); err != nil {
		return derp.Wrap(err, "service.Search.Save", "Error validating Search", searchResult)
	}*/

	// Save the searchResult to the database
	if err := service.collection.Save(searchResult, note); err != nil {
		return derp.Wrap(err, "service.Search.Save", "Error saving Search", searchResult, note)
	}

	for _, tag := range searchResult.Tags {
		if err := service.searchTagService.Upsert(tag); err != nil {
			return derp.Wrap(err, "service.Search.Save", "Error saving SearchTag", searchResult, tag)
		}
	}

	return nil
}

// Delete removes an Search from the database (HARD DELETE)
func (service *Search) Delete(searchResult *model.SearchResult, note string) error {

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

func (service *Search) RangeByTags(tags ...string) (iter.Seq[model.SearchResult], error) {
	return service.Range(exp.In("tags", tags))
}

func (service *Search) LoadByURL(url string, searchResult *model.SearchResult) error {
	return service.Load(exp.Equal("url", url), searchResult)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *Search) Upsert(searchResult model.SearchResult) error {

	// First, try to load the original Search
	original := model.NewSearchResult()

	if err := service.LoadByURL(searchResult.URL, &original); !derp.NilOrNotFound(err) {
		return derp.Wrap(err, "service.Search.Upsert", "Error loading Search", searchResult)
	} else if err == nil {
		original.Update(searchResult)
	} else {
		original = searchResult
	}

	// Update the original Search with the new values
	original.Update(searchResult)
	comment := "added"

	if !original.IsNew() {
		comment = "updated"
	}

	if err := service.Save(&original, comment); err != nil {
		return derp.Wrap(err, "service.Search.Add", "Error adding Search", searchResult)
	}

	return nil
}

func (service *Search) DeleteByURL(url string) error {
	searchResult := model.NewSearchResult()

	if err := service.LoadByURL(url, &searchResult); err != nil {

		if derp.NotFound(err) {
			return nil
		}

		return derp.Wrap(err, "service.Search.DeleteByURL", "Error loading Search", url)
	}

	return service.Delete(&searchResult, "deleted from search index")
}

// Shuffle updates the "shuffle" field for all SearchResults that match the provided tags
func (service *Search) Shuffle(tags ...string) error {

	const location = "service.Search.Shuffle"

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
}
