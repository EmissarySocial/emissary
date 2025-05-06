package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Connection manages all interactions with the Connection collection
type Connection struct {
	collection      data.Collection
	providerService *Provider
}

// NewConnection returns a fully populated Connection service
func NewConnection() Connection {
	return Connection{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Connection) Refresh(collection data.Collection, providerService *Provider) {
	service.collection = collection
	service.providerService = providerService
}

// Close stops any background processes controlled by this service
func (service *Connection) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Connection) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Connection) Query(criteria exp.Expression, options ...option.Option) ([]model.Connection, error) {
	result := make([]model.Connection, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Connections who match the provided criteria
func (service *Connection) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Connection from the database
func (service *Connection) Load(criteria exp.Expression, result *model.Connection) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Connection.Load", "Error loading Connection", criteria)
	}

	return nil
}

// Save adds/updates an Connection in the database
func (service *Connection) Save(group *model.Connection, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(group); err != nil {
		return derp.Wrap(err, "service.Connection.Save", "Error validating Connection", group)
	}

	// Save the value to the database
	if err := service.collection.Save(group, note); err != nil {
		return derp.Wrap(err, "service.Connection.Save", "Error saving Connection", group, note)
	}

	return nil
}

// Delete removes an Connection from the database (virtual delete)
func (service *Connection) Delete(group *model.Connection, note string) error {

	if err := service.collection.Delete(group, note); err != nil {
		return derp.Wrap(err, "service.Connection.Delete", "Error deleting Connection", group, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Connection) ObjectType() string {
	return "Connection"
}

// New returns a fully initialized model.Connection as a data.Object.
func (service *Connection) ObjectNew() data.Object {
	result := model.NewConnection()
	return &result
}

func (service *Connection) ObjectID(object data.Object) primitive.ObjectID {

	if group, ok := object.(*model.Connection); ok {
		return group.ConnectionID
	}

	return primitive.NilObjectID
}

func (service *Connection) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Connection) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewConnection()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Connection) ObjectSave(object data.Object, comment string) error {
	if group, ok := object.(*model.Connection); ok {
		return service.Save(group, comment)
	}
	return derp.InternalError("service.Connection.ObjectSave", "Invalid Object Type", object)
}

func (service *Connection) ObjectDelete(object data.Object, comment string) error {
	if group, ok := object.(*model.Connection); ok {
		return service.Delete(group, comment)
	}
	return derp.InternalError("service.Connection.ObjectDelete", "Invalid Object Type", object)
}

func (service *Connection) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Connection", "Not Authorized")
}

func (service *Connection) Schema() schema.Schema {
	return schema.New(model.ConnectionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Connection) QueryAll(options ...option.Option) ([]model.Connection, error) {
	return service.Query(exp.All(), options...)
}

func (service *Connection) AllAsMap() mapof.Object[model.Connection] {
	result := make(mapof.Object[model.Connection])

	if query, err := service.QueryAll(); err == nil {
		for _, connection := range query {
			result[connection.ProviderID] = connection
		}
	}

	return result
}

func (service *Connection) LoadActiveByType(typeID string, connection *model.Connection) error {

	criteria := exp.Equal("type", typeID).AndEqual("active", true)

	if err := service.Load(criteria, connection); err != nil {
		return derp.Wrap(err, "service.Connection.LoadByType", "Error loading Connection", typeID)
	}

	return nil
}

// LoadByProvider loads a Connection that matches the given provider.
func (service *Connection) LoadByProvider(providerID string, connection *model.Connection) error {

	criteria := exp.Equal("providerId", providerID)

	if err := service.Load(criteria, connection); err != nil {
		return derp.Wrap(err, "service.Connection.LoadByProvider", "Error loading Connection", providerID)
	}

	return nil
}

// LoadOrCreateByProvider loads a Connection that matches the given provider.  If no Connection is found, a new one is created.
func (service *Connection) LoadOrCreateByProvider(providerID string) (model.Connection, error) {

	result := model.NewConnection()

	criteria := exp.Equal("providerId", providerID)

	err := service.Load(criteria, &result)
	if err == nil {
		return result, nil
	}

	if derp.NotFound(err) {
		result.ProviderID = providerID
		return result, nil
	}

	return result, derp.Wrap(err, "service.Connection.LoadOrCreateByProvider", "Error loading Connection", providerID)
}
