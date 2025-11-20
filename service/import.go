package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Import service helps exports user data to another server
type Import struct {
	activityService ActivityStream
}

// NewImport returns a fully populated Import service
func NewImport(activityService ActivityStream) Import {
	return Import{
		activityService: activityService,
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Import) collection(session data.Session) data.Collection {
	return session.Collection("Import")
}

func (service *Import) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Import, error) {
	result := make([]model.Import, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Count returns the number of records that match the provided criteria
func (service *Import) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Imports who match the provided criteria
func (service *Import) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Import records that match the provided criteria
func (service *Import) Range(session data.Session, criteria exp.Expression, options ...option.Option) iter.Seq[model.Import] {

	return func(yield func(model.Import) bool) {

		// Retrieve the Imports from the database
		records, err := service.List(session, criteria, options...)

		// Soft fail.  Report, but do not crash.
		if err != nil {
			derp.Report(derp.Wrap(err, "service.Import.Range", "Unable to create iterator", criteria))
			return
		}

		defer derp.ReportFunc(records.Close)

		// Yield each import to the caller one-by-one
		for record := model.NewImport(); records.Next(&record); record = model.NewImport() {
			if !yield(record) {
				return
			}
		}
	}
}

// Load retrieves an Import from the database
func (service *Import) Load(session data.Session, criteria exp.Expression, record *model.Import) error {

	if err := service.collection(session).Load(notDeleted(criteria), record); err != nil {
		return derp.Wrap(err, "service.Import.Load", "Unable to load Import", criteria)
	}

	return nil
}

// Save adds/updates an Import in the database
func (service *Import) Save(session data.Session, record *model.Import, note string) error {

	const location = "service.Import.Save"

	spew.Dump(location, record)

	// Validate the value before saving
	if err := service.Schema().Validate(record); err != nil {
		return derp.Wrap(err, location, "Invalid Import record", record)
	}

	// Execute state changes
	service.calcStateChange(record)
	spew.Dump("after-state", record)

	// Save the import to the database
	if err := service.collection(session).Save(record, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Import", record, note)
	}

	return nil
}

// Delete removes an Import from the database (virtual delete)
func (service *Import) Delete(session data.Session, record *model.Import, note string) error {

	const location = "service.Import.Delete"

	// Delete this Import
	if err := service.collection(session).Delete(record, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Import", record, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Import) ObjectType() string {
	return "Import"
}

// New returns a fully initialized model.Import as a data.Object.
func (service *Import) ObjectNew() data.Object {
	result := model.NewImport()
	return &result
}

// ObjectID returns the ID of a record object
func (service *Import) ObjectID(object data.Object) primitive.ObjectID {

	if record, ok := object.(*model.Import); ok {
		return record.ImportID
	}

	return primitive.NilObjectID
}

func (service *Import) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Import) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewImport()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Import) ObjectSave(session data.Session, object data.Object, note string) error {
	if record, ok := object.(*model.Import); ok {
		return service.Save(session, record, note)
	}
	return derp.InternalError("service.Import.ObjectSave", "Invalid object type", object)
}

func (service *Import) ObjectDelete(session data.Session, object data.Object, note string) error {
	if record, ok := object.(*model.Import); ok {
		return service.Delete(session, record, note)
	}
	return derp.InternalError("service.Import.ObjectDelete", "Invalid object type", object)
}

func (service *Import) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Import.ObjectUserCan", "Not Authorized")
}

func (service *Import) Schema() schema.Schema {
	return schema.New(model.ImportSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Import) QueryByUser(session data.Session, userID primitive.ObjectID) ([]model.Import, error) {
	criteria := exp.Equal("userId", userID)
	return service.Query(session, criteria)
}

func (service *Import) LoadByID(session data.Session, userID primitive.ObjectID, importID primitive.ObjectID, record *model.Import) error {
	criteria := exp.Equal("_id", importID).AndEqual("userId", userID)
	return service.Load(session, criteria, record)
}

func (service *Import) LoadByToken(session data.Session, userID primitive.ObjectID, token string, record *model.Import) error {

	importID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Import.LoadByToken", "Import token must be a valid ObjectID", token)
	}

	return service.LoadByID(session, userID, importID, record)
}

/******************************************
 * State Machine
 ******************************************/

func (service *Import) calcStateChange(record *model.Import) {

	switch record.StateID {

	case model.ImportStateDoAuthorize:
		service.doAuthorize(record)

	case model.ImportStateDoImport:
		service.doImport(record)

	case model.ImportStateDoMove:
		service.doMove(record)
	}
}

// doAuthorize manages the transient state change from "DO-AUTHORIZE"
// to "AUTHORIZING".
func (service *Import) doAuthorize(record *model.Import) {

	// Find the remote actor identified as the Source account
	client := service.activityService.Client()
	actor, err := client.Load(record.SourceID)

	if err != nil {
		record.StateID = model.ImportStateAuthorizationError
		record.StateDescription = "The account you entered (" + record.SourceID + ") could not be found. Please enter a different account."
		spew.Dump(record)
		return
	}

	// RULE: Require that the remote actor is a "Person"
	if actor.Type() != vocab.ActorTypePerson {
		record.StateID = model.ImportStateAuthorizationError
		record.StateDescription = "The account you entered (" + record.SourceID + ") is not valid because it is a '" + actor.Type() + "' type record. You can only import from 'Person' accounts."
		spew.Dump(record)
		return
	}

	// Locate the migration endpoint
	record.SourceOAuthURL = actor.Endpoints().Get(vocab.EndpointOAuthMigration).String()

	if record.SourceOAuthURL == "" {
		record.StateID = model.ImportStateAuthorizationError
		record.StateDescription = "The account you entered (" + record.SourceID + ") does not support account migration.  Actors must define an OAuth migration endpoint to be compatible."
		spew.Dump(record)
		return
	}

	// SUCCESS (for now)
	record.StateID = model.ImportStateAuthorizing
	record.StateDescription = ""
	spew.Dump(record)
}

// doImport manages the transient state change from "DO-IMPORT"
// to "IMPORTING"
func (service *Import) doImport(record *model.Import) {
}

// doMove manages the transient state change from "DO-MOVE"
// to "MOVING"
func (service *Import) doMove(record *model.Import) {
}
