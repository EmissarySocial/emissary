package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery defines a service that manages all searchable tags in a domain.
type SearchQuery struct {
	collection       data.Collection
	searchTagService *SearchTag
}

// NewSearchQuery returns a fully initialized SearchQuery service
func NewSearchQuery() SearchQuery {
	return SearchQuery{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchQuery) Refresh(collection data.Collection, searchTagService *SearchTag) {
	service.collection = collection
	service.searchTagService = searchTagService
}

// Close stops any background processes controlled by this service
func (service *SearchQuery) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchQuery) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe SearchQuerys that match the provided criteria
func (service *SearchQuery) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchQuery, error) {
	result := make([]model.SearchQuery, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the SearchQuerys that match the provided criteria
func (service *SearchQuery) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an SearchQuery from the database
func (service *SearchQuery) Load(criteria exp.Expression, searchQuery *model.SearchQuery) error {

	if err := service.collection.Load(notDeleted(criteria), searchQuery); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Load", "Error loading SearchQuery", criteria)
	}

	return nil
}

// Save adds/updates an SearchQuery in the database
func (service *SearchQuery) Save(searchQuery *model.SearchQuery, note string) error {

	const location = "service.SearchQuery.Save"

	if len(searchQuery.Original) > 128 {
		return derp.New(derp.CodeBadRequestError, location, "SearchQuery.Original is too long", searchQuery)
	}

	// Split the query into tags and remainder
	tags, remainder := parse.HashtagsAndRemainder(searchQuery.Original)

	_, tagValues, err := service.searchTagService.NormalizeTags(tags...)

	if err != nil {
		return derp.Wrap(err, location, "Error normalizing tags", searchQuery)
	}

	// Update values
	searchQuery.TagValues = tagValues
	searchQuery.Remainder = strings.TrimSpace(remainder)

	// Save the searchQuery to the database
	if err := service.collection.Save(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Save", "Error saving SearchQuery", searchQuery, note)
	}

	return nil
}

// Delete removes an SearchQuery from the database (virtual delete)
func (service *SearchQuery) Delete(searchQuery *model.SearchQuery, note string) error {

	// Delete this SearchQuery
	if err := service.collection.Delete(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Delete", "Error deleting SearchQuery", searchQuery, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *SearchQuery) LoadByToken(token string, searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.LoadByToken"

	// Parse the token as an ID
	searchQueryID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Error converting token to ObjectID", token)
	}

	// Query the database
	criteria := exp.Equal("_id", searchQueryID)
	return service.Load(criteria, searchQuery)
}

/******************************************
 * Custom Actions
 ******************************************/
