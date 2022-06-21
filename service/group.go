package service

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/queries"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group manages all interactions with the Group collection
type Group struct {
	collection data.Collection
}

// NewGroup returns a fully populated Group service
func NewGroup(collection data.Collection) Group {
	return Group{
		collection: collection,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Groups who match the provided criteria
func (service *Group) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Group from the database
func (service *Group) Load(criteria exp.Expression, result *model.Group) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Group", "Error loading Group", criteria)
	}

	return nil
}

// Save adds/updates an Group in the database
func (service *Group) Save(user *model.Group, note string) error {

	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.Group", "Error saving Group", user, note)
	}

	return nil
}

// Delete removes an Group from the database (virtual delete)
func (service *Group) Delete(user *model.Group, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.Group", "Error deleting Group", user, note)
	}

	// TODO: Also remove connections to Users that still use this Group
	// TODO: Also remove connections to Streams that still use this Group

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Group as a data.Object.
func (service *Group) ObjectNew() data.Object {
	result := model.NewGroup()
	return &result
}

func (service *Group) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Group) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewGroup()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Group) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.Group), comment)
}

func (service *Group) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.Group), comment)
}

func (service *Group) Debug() maps.Map {
	return maps.Map{
		"service": "Group",
	}
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// LoadByID loads a single model.Group object that matches the provided userID
func (service *Group) LoadByID(groupID primitive.ObjectID, result *model.Group) error {
	criteria := exp.Equal("_id", groupID)
	return service.Load(criteria, result)
}

func (service *Group) ListByIDs(groupIDs ...primitive.ObjectID) ([]model.Group, error) {

	result := make([]model.Group, len(groupIDs)+1)

	// If there are no groupIDs, then there's nothing to query.  Let's keep it simple, yes?
	if len(groupIDs) == 0 {
		return result, nil
	}

	// Build the criteria from the list of GroupIDs
	criteria := exp.Empty()

	for _, groupID := range groupIDs {
		criteria = criteria.Or(exp.Equal("_id", groupID))
	}

	// Query the database for all matching groups
	it, err := service.List(criteria, option.SortAsc("label"))

	if err != nil {
		return nil, derp.Wrap(err, "service.Group.ListbyIDs", "Error executing query", criteria)
	}

	// Read the iterator into a result array
	index := 0

	for it.Next(&(result[index])) {
		index++
	}

	// Trim the results just in case one of the groupIDs was not valid.
	// result = result[:index]

	return result, nil
}

// LoadByGroupname loads a single model.Group object that matches the provided token
func (service *Group) LoadByToken(token string, result *model.Group) error {

	// If the token *looks* like an ObjectID then try that first.  If it works, then return in triumph
	if userID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(userID, result); err == nil {
			return nil
		}
	}

	// Otherwise, use the token as a username
	criteria := exp.Equal("token", token)
	return service.Load(criteria, result)
}

// ListByGroup returns all users that match a provided group name
func (service *Group) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}

func (service *Group) ListAsOptions() ([]form.OptionCode, error) {

	it, err := service.List(exp.All(), option.SortAsc("label"))

	if err != nil {
		return nil, derp.Wrap(err, "service.Group.ListAsOptions", "Error listing Groups")
	}

	result := make([]form.OptionCode, 0)

	var group model.Group
	for it.Next(&group) {
		result = append(result, form.OptionCode{Label: group.Label, Value: group.GroupID.Hex()})
	}

	return result, nil
}

// Count returns the number of (non-deleted) records in the User collection
func (service *Group) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}
