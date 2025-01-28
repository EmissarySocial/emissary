package service

import (
	"net/url"
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

	// Split the query, and Normalize tags and remainder
	if err := service.parseHashtags(searchQuery); err != nil {
		return derp.Wrap(err, location, "Error normalizing tags", searchQuery)
	}

	// RULE: Do not allow global searches here.
	if searchQuery.IsEmpty() {
		return derp.New(derp.CodeBadRequestError, location, "SearchQuery is empty", searchQuery)
	}

	wasNew := searchQuery.IsNew()

	// Save the searchQuery to the database
	if err := service.collection.Save(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Save", "Error saving SearchQuery", searchQuery, note)
	}

	if wasNew {
		// TODO: Add a queue task to try to delete this SearchQuery if it hasn't been subscribed after 1 day
	}

	return nil
}

func (service *SearchQuery) Upsert(searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.Upsert"

	if err := service.parseHashtags(searchQuery); err != nil {
		return derp.Wrap(err, location, "Error validating query string", searchQuery.Original)
	}

	if searchQuery.IsEmpty() {
		return derp.New(derp.CodeBadRequestError, location, "SearchQuery is empty", searchQuery)
	}

	if err := service.LoadByTagsAndRemainder(searchQuery.TagValues, searchQuery.Remainder, searchQuery); err != nil {

		if derp.NotFound(err) {

			if err := service.Save(searchQuery, "Upsert"); err != nil {
				return derp.Wrap(err, location, "Error saving SearchQuery", searchQuery)
			}

			return nil
		}

		return derp.Wrap(err, location, "Error loading SearchQuery", searchQuery)
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

func (service *SearchQuery) LoadByTagsAndRemainder(tags []string, remainder string, searchQuery *model.SearchQuery) error {
	criteria := exp.InAll("tagValues", tags).And(exp.Equal("remainder", remainder))
	return service.Load(criteria, searchQuery)
}

func (service *SearchQuery) LoadByQueryString(queryValues url.Values, searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.LoadByQueryString"

	// If we have a searchID token, then try to use it first.
	if token := queryValues.Get("id"); token != "" {
		if err := service.LoadByToken(token, searchQuery); err != nil {
			return derp.Wrap(err, location, "Error loading SearchQuery by token", token)
		}
	}

	// Fall through means there's no token, or a deleted token.
	if query := queryValues.Get("q"); query != "" {

		searchQuery.Original = query

		if err := service.parseHashtags(searchQuery); err != nil {
			return derp.Wrap(err, location, "Error normalizing tags", searchQuery)
		}

		if err := service.LoadByTagsAndRemainder(searchQuery.TagValues, searchQuery.Remainder, searchQuery); err == nil {
			return nil
		}

		if err := service.Upsert(searchQuery); err != nil {
			return derp.Wrap(err, location, "Error upserting SearchQuery", query)
		}

		return nil
	}

	return derp.NewBadRequestError(location, "No search query provided", queryValues)
}

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

// parseHashtags splits the search query into tags and remainder,
// then normalizes the tags by removing blocked tags and trimming the remainder
func (service *SearchQuery) parseHashtags(searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.parseHashtags"

	// Split the original query into tags and remainder
	tags, remainder := parse.HashtagsAndRemainder(searchQuery.Original)

	// Normalize the tags
	_, tags, err := service.searchTagService.NormalizeTags(tags...)

	if err != nil {
		return derp.Wrap(err, location, "Error normalizing tags", searchQuery)
	}

	// Update the SearchQuery with the normalized values
	searchQuery.TagValues = tags
	searchQuery.Remainder = strings.TrimSpace(remainder)

	return nil
}
