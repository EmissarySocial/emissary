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
	"github.com/benpate/rosetta/mapof"
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

// Query returns an slice of allthe Searchs that match the provided criteria
func (service *SearchResult) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchResult, error) {
	result := make([]model.SearchResult, 0)
	err := service.collection.Query(&result, criteria, options...)

	return result, err
}

// List returns an iterator containing all of the Searchs that match the provided criteria
func (service *SearchResult) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(criteria, options...)
}

func (service *SearchResult) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchResult], error) {
	it, err := service.collection.Iterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Search.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchResult), nil
}

// Load retrieves an Search from the database
func (service *SearchResult) Load(criteria exp.Expression, searchResult *model.SearchResult) error {

	if err := service.collection.Load(criteria, searchResult); err != nil {
		return derp.Wrap(err, "service.Search.Load", "Error loading Search", criteria)
	}

	return nil
}

// Save adds/updates an Search in the database
func (service *SearchResult) Save(searchResult *model.SearchResult, note string) error {

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

	for _, tagName := range searchResult.TagNames {
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

func (service *SearchResult) RangeByTags(tags ...string) (iter.Seq[model.SearchResult], error) {
	return service.Range(exp.In("tags", tags))
}

func (service *SearchResult) LoadByURL(url string, searchResult *model.SearchResult) error {
	return service.Load(exp.Equal("url", url), searchResult)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *SearchResult) Sync(searchResult model.SearchResult) error {

	if searchResult.IsDeleted() {
		return service.DeleteByURL(searchResult.URL)
	}

	return service.upsert(searchResult)
}

// upsert adds or updates a SearchResult in the database
func (service *SearchResult) upsert(searchResult model.SearchResult) error {

	// First, try to load the original Search
	original := model.NewSearchResult()

	err := service.LoadByURL(searchResult.URL, &original)

	if err == nil {
		original.Update(searchResult)
	} else if derp.NotFound(err) {
		original = searchResult
	} else {
		return derp.Wrap(err, "service.Search.Upsert", "Error loading Search", searchResult)
	}

	comment := iif(original.IsNew(), "added", "updated")
	if err := service.Save(&original, comment); err != nil {
		return derp.Wrap(err, "service.Search.Add", "Error adding Search", searchResult)
	}

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

func (service *SearchResult) UnmarshalMap(original map[string]any) model.SearchResult {

	value := mapof.Any(original)
	searchResult := model.NewSearchResult()
	searchResult.Type = value.GetString("type")
	searchResult.URL = value.GetString("url")
	searchResult.Name = value.GetString("name")
	searchResult.AttributedTo = value.GetString("attributedTo")
	searchResult.Summary = value.GetString("summary")
	searchResult.IconURL = value.GetString("icon")
	searchResult.Rank = value.GetInt64("rank")
	searchResult.Shuffle = value.GetInt64("shuffle")
	searchResult.FullText = value.GetString("fullText")

	// Special handling for tags
	tagNames, tagValues, err := service.searchTagService.NormalizeTags(value.GetSliceOfString("tagNames")...)
	derp.Report(derp.Wrap(err, "service.Search.UnmarshalMap", "Error normalizing tags", value))

	searchResult.TagNames = tagNames
	searchResult.TagValues = tagValues

	return searchResult
}
