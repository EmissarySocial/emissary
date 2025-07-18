package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group manages all interactions with the Group collection
type Group struct {
	collection data.Collection
}

// NewGroup returns a fully populated Group service
func NewGroup() Group {
	return Group{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Group) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Group) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Group) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Group) Query(criteria exp.Expression, options ...option.Option) ([]model.Group, error) {
	result := make([]model.Group, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Groups who match the provided criteria
func (service *Group) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Group from the database
func (service *Group) Load(criteria exp.Expression, result *model.Group) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Group.Load", "Error loading Group", criteria)
	}

	return nil
}

// Save adds/updates an Group in the database
func (service *Group) Save(group *model.Group, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(group); err != nil {
		return derp.Wrap(err, "service.Group.Save", "Error validating Group", group)
	}

	// Save the value to the database
	if err := service.collection.Save(group, note); err != nil {
		return derp.Wrap(err, "service.Group.Save", "Error saving Group", group, note)
	}

	return nil
}

// Delete removes an Group from the database (virtual delete)
func (service *Group) Delete(group *model.Group, note string) error {

	if err := service.collection.Delete(group, note); err != nil {
		return derp.Wrap(err, "service.Group.Delete", "Error deleting Group", group, note)
	}

	// TODO: HIGH: Also remove connections to Users that still use this Group
	// TODO: HIGH: Also remove connections to Streams that still use this Group

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Group) ObjectType() string {
	return "Group"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Group) ObjectNew() data.Object {
	result := model.NewGroup()
	return &result
}

func (service *Group) ObjectID(object data.Object) primitive.ObjectID {

	if group, ok := object.(*model.Group); ok {
		return group.GroupID
	}

	return primitive.NilObjectID
}

func (service *Group) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Group) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewGroup()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Group) ObjectSave(object data.Object, comment string) error {
	if group, ok := object.(*model.Group); ok {
		return service.Save(group, comment)
	}
	return derp.InternalError("service.Group.ObjectSave", "Invalid Object Type", object)
}

func (service *Group) ObjectDelete(object data.Object, comment string) error {
	if group, ok := object.(*model.Group); ok {
		return service.Delete(group, comment)
	}
	return derp.InternalError("service.Group.ObjectDelete", "Invalid Object Type", object)
}

func (service *Group) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Group", "Not Authorized")
}

func (service *Group) Schema() schema.Schema {
	return schema.New(model.GroupSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID loads a single model.Group object that matches the provided groupID
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

// LoadByToken loads a single Group object that matches the provided token
func (service *Group) LoadByToken(token string, result *model.Group) error {

	// Trim whitespace around the token
	token = strings.Trim(token, " ")

	// If the token *looks* like an ObjectID then try that first.  If it works, then return in triumph
	if groupID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(groupID, result); err == nil {
			return nil
		}
	}

	// Otherwise, use the token as a groupID
	criteria := exp.Equal("token", token)
	return service.Load(criteria, result)
}

// ListByGroup returns all groups that match a provided group name
func (service *Group) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}

func (service *Group) ListAsOptions() []form.LookupCode {

	result := make([]form.LookupCode, 0)

	it, err := service.List(exp.All(), option.SortAsc("label"))

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Group.ListAsOptions", "Error listing Groups"))
		return result
	}

	var group model.Group
	for it.Next(&group) {
		result = append(result, form.LookupCode{
			Label: group.Label,
			Value: group.GroupID.Hex(),
			Icon:  "people",
		})
	}

	return result
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *Group) Startup(theme *model.Theme) error {

	// Try to count the number of existing groups in the database
	count, err := service.Count(exp.All())

	if err != nil {
		return derp.Wrap(err, "service.Theme.Startup", "Error counting groups")
	}

	// If there are already groups in the database, then don't make any changes.
	if count > 0 {
		return nil
	}

	// Create groups
	groupSchema := schema.New(model.GroupSchema())

	for _, data := range theme.StartupGroups {
		group := model.NewGroup()

		if err := groupSchema.SetAll(&group, data); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Unable to set group data", data))
			continue
		}

		if err := service.Save(&group, "Created by Startup"); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Unable to save group", group))
			continue
		}
	}

	return nil
}
