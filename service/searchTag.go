package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag defines a service that manages all searchable tags in a domain.
type SearchTag struct {
	collection data.Collection
	host       string
}

// NewSearchTag returns a fully initialized SearchTag service
func NewSearchTag() SearchTag {
	return SearchTag{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchTag) Refresh(collection data.Collection, host string) {
	service.collection = collection
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchTag) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchTag) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe SearchTags that match the provided criteria
func (service *SearchTag) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchTag, error) {
	result := make([]model.SearchTag, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the SearchTags that match the provided criteria
func (service *SearchTag) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an SearchTag from the database
func (service *SearchTag) Load(criteria exp.Expression, searchTag *model.SearchTag) error {

	if err := service.collection.Load(notDeleted(criteria), searchTag); err != nil {
		return derp.Wrap(err, "service.SearchTag.Load", "Error loading SearchTag", criteria)
	}

	return nil
}

// LoadWithOptions retrieves a single SearchTag from the database, with additional options
func (service *SearchTag) LoadWithOptions(criteria exp.Expression, searchTag *model.SearchTag, options ...option.Option) error {

	options = append(options, option.MaxRows(1))

	results, err := service.Query(criteria, options...)

	if err != nil {
		return derp.Wrap(err, "service.SearchTag.LoadByName", "Error loading SearchTag", criteria)
	}

	if len(results) == 0 {
		return derp.NewNotFoundError("service.SearchTag.LoadByName", "SearchTag not found", criteria)
	}

	*searchTag = results[0]

	return nil
}

// Save adds/updates an SearchTag in the database
func (service *SearchTag) Save(searchTag *model.SearchTag, note string) error {

	// Calculate the searchable value for this SearchTag
	searchTag.Value = model.ToToken(searchTag.Name)

	// Validate the value before saving
	if err := service.Schema().Validate(searchTag); err != nil {
		return derp.Wrap(err, "service.SearchTag.Save", "Error validating SearchTag", searchTag)
	}

	// Save the searchTag to the database
	if err := service.collection.Save(searchTag, note); err != nil {
		return derp.Wrap(err, "service.SearchTag.Save", "Error saving SearchTag", searchTag, note)
	}

	return nil
}

// Delete removes an SearchTag from the database (virtual delete)
func (service *SearchTag) Delete(searchTag *model.SearchTag, note string) error {

	// Delete this SearchTag
	if err := service.collection.Delete(searchTag, note); err != nil {
		return derp.Wrap(err, "service.SearchTag.Delete", "Error deleting SearchTag", searchTag, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *SearchTag) ObjectType() string {
	return "SearchTag"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *SearchTag) ObjectNew() data.Object {
	result := model.NewSearchTag()
	return &result
}

func (service *SearchTag) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.SearchTag); ok {
		return mention.SearchTagID
	}

	return primitive.NilObjectID
}

func (service *SearchTag) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *SearchTag) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *SearchTag) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewSearchTag()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *SearchTag) ObjectSave(object data.Object, comment string) error {
	if searchTag, ok := object.(*model.SearchTag); ok {
		return service.Save(searchTag, comment)
	}
	return derp.NewInternalError("service.SearchTag.ObjectSave", "Invalid Object Type", object)
}

func (service *SearchTag) ObjectDelete(object data.Object, comment string) error {
	if searchTag, ok := object.(*model.SearchTag); ok {
		return service.Delete(searchTag, comment)
	}
	return derp.NewInternalError("service.SearchTag.ObjectDelete", "Invalid Object Type", object)
}

func (service *SearchTag) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.SearchTag", "Not Authorized")
}

func (service *SearchTag) Schema() schema.Schema {
	return schema.New(model.SearchTagSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *SearchTag) LoadByID(searchTagID primitive.ObjectID, searchTag *model.SearchTag) error {
	criteria := exp.Equal("_id", searchTagID)
	return service.Load(criteria, searchTag)
}

func (service *SearchTag) LoadByValue(value string, searchTag *model.SearchTag) error {
	criteria := exp.Equal("value", model.ToToken(value))
	return service.LoadWithOptions(criteria, searchTag, option.CaseSensitive(false))
}

// Upsert verifies that a SearchTag exists in the database, and creates it if it does not.
func (service *SearchTag) Upsert(tagName string) error {

	searchTag := model.NewSearchTag()
	value := model.ToToken(tagName)

	// Try to find the SearchTag in the database
	err := service.LoadByValue(value, &searchTag)

	// If it exists, then we're done
	if err == nil {
		return nil
	}

	// If "not found" then create a new SearchTag``
	if derp.NotFound(err) {

		// Set default values for the new SearchTag
		searchTag.Name = tagName

		if err := service.Save(&searchTag, "Found New SearchTag"); err != nil {
			return derp.Wrap(err, "service.SearchTag.Upsert", "Error saving SearchTag", value)
		}

		return nil
	}

	// Otherwise, return the error to the caller. (This should never happen)
	return derp.Wrap(err, "service.SearchTag.Upsert", "Error loading SearchTag", value)
}

// ListGroups returns a distinct list of all the groups that are used by SearchTags
func (service *SearchTag) ListGroups() []form.LookupCode {

	const location = "service.SearchTag.ListGroups"

	groups, err := queries.SearchTags_Groups(service.collection)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error reading distinct groups"))
		return []form.LookupCode{}
	}

	result := make([]form.LookupCode, len(groups))

	for index, group := range groups {
		result[index] = form.LookupCode{
			Value: group,
			Label: group,
		}
	}

	return result
}

// FindAllowedTags returns a list of tag VALUES that match the query string
func (service *SearchTag) FindAllowedTags(query string) ([]string, error) {

	const location = "service.SearchTag.FindAllowedTags"

	// Split tags into a slice and normalize tag names
	tagValues := parse.Split(query)
	tagValues = slice.Map(tagValues, model.ToToken)

	// Query the database for ALLOWED and FEATURED tags that match
	criteria := exp.In("value", tagValues).
		AndIn("stateId", []int{model.SearchTagStateAllowed, model.SearchTagStateFeatured})

	searchTags, err := service.Query(criteria, option.Fields("value"))

	if err != nil {
		return []string{}, derp.Wrap(err, location, "Error querying SearchTags", criteria)
	}

	// Map the results into a single string value
	result := slice.Map(searchTags, func(tag model.SearchTag) string {
		return tag.Value
	})

	return result, nil
}

// QueryByValue returns all tags in a list
func (service *SearchTag) QueryByValue(values []string, options ...option.Option) (sliceof.Object[model.SearchTag], error) {
	criteria := exp.In("value", values)
	return service.Query(criteria, options...)
}

/******************************************
 * Custom Actions
 ******************************************/

// NormalizeTags takes a list of tag names and verifies it against tags in the database.
// Tags using canonical names will be returned. Blocked tags will not be included.
// If a tag does not exist in the database, then the provided name will be used.
func (service *SearchTag) NormalizeTags(tagNames ...string) (sliceof.String, sliceof.String, error) {

	const location = "service.SearchTag.NormalizeTags"

	// Sort so we can traverse both slices simultaneously.
	slices.Sort(tagNames)

	// use canonical values for all tag names
	tagValues := slice.Map(tagNames, model.ToToken)

	// Retrieve all matching tags (sorted by value)
	dbTags, err := service.QueryByValue(tagValues, option.SortAsc("value"))

	if err != nil {
		return sliceof.NewString(), sliceof.NewString(), derp.Wrap(err, location, "Error querying existing tags")
	}

	// Initialize Result values
	resultNames := make(sliceof.String, 0, len(tagNames))
	resultValues := make(sliceof.String, 0, len(tagNames))

	// Loop through tagNames AND tagValues
	for tagIndex := range tagNames {

		tagName := tagNames[tagIndex]
		tagValue := tagValues[tagIndex]

		// Search for the tagValue in the database results
		dbTag, found := dbTags.Find(func(tag model.SearchTag) bool {
			return tag.Value == tagValue
		})

		if found {

			// Add non-blocked tags to the result
			if dbTag.StateID != model.SearchTagStateBlocked {
				resultNames = append(resultNames, dbTag.Name)
				resultValues = append(resultValues, dbTag.Value)
			}

			continue
		}

		// Add new tags to the result (these will be "Waiting")
		resultNames = append(resultNames, tagName)
		resultValues = append(resultValues, tagValue)
	}

	// Sort the values
	slices.Sort(resultValues)

	// Success?!?!?
	return resultNames, resultValues, nil
}
