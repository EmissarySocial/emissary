package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Annotation manages all interactions with the Annotation collection
type Annotation struct {
	activityService   *ActivityStream
	importItemService *ImportItem
}

// NewAnnotation returns a fully populated Annotation service
func NewAnnotation() Annotation {
	return Annotation{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Annotation) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.importItemService = factory.ImportItem()
}

// Close stops any background processes controlled by this service
func (service *Annotation) Close() {
	// No background processes to stop
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Annotation collection for the provided database session
func (service *Annotation) collection(session data.Session) data.Collection {
	return session.Collection("Annotation")
}

// Count returns the number of records that match the provided criteria
func (service *Annotation) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns every Annotation that matches the provided criteria
func (service *Annotation) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Annotation, error) {
	result := make([]model.Annotation, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Annotations that match the provided criteria
func (service *Annotation) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Annotations that match the provided criteria
func (service *Annotation) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Annotation], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.User.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewAnnotation), nil
}

// Load retrieves an Annotation from the database
func (service *Annotation) Load(session data.Session, criteria exp.Expression, result *model.Annotation) error {

	const location = "service.Annotation.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Loading Annotation", criteria)
	}

	return nil
}

// Save adds/updates an Annotation in the database, keeping one Annotation per User and URL
func (service *Annotation) Save(session data.Session, annotation *model.Annotation, note string) error {
	client := service.activityService.UserClient(annotation.UserID)
	return service.save(session, client, annotation, note)
}

// save describes the Annotation using the provided client, then writes it to the database
func (service *Annotation) save(session data.Session, client streams.Client, annotation *model.Annotation, note string) error {

	const location = "service.Annotation.save"

	// Copy values from the annotated document
	if err := describeAnnotation(client, annotation); err != nil {

		// RULE: A new annotation needs a reachable document. An existing one keeps its stored
		// description, so its note can still be edited after the document disappears.
		if annotation.IsNew() {
			return derp.Wrap(err, location, "Loading annotated document", annotation.URL)
		}
	}

	// Validate the value before saving
	if _, err := service.Schema().Validate(annotation); err != nil {
		return derp.Wrap(err, location, "Validating Annotation", annotation)
	}

	// Save the value to the database
	if err := service.collection(session).Save(annotation, note); err != nil {

		// RULE: A lost race trips idx_Annotation_User_URL_Unique, which data-mongo reports as a
		// Conflict. Fold onto the winner's record and retry once, so both saves land on one row.
		if !derp.IsConflict(err) {
			return derp.Wrap(err, location, "Saving Annotation", annotation, note)
		}

		if err := service.adoptExisting(session, annotation); err != nil {
			return derp.Wrap(err, location, "Loading the Annotation that won the race", annotation.URL)
		}

		if err := service.collection(session).Save(annotation, note); err != nil {
			return derp.Wrap(err, location, "Saving Annotation after conflict", annotation, note)
		}
	}

	// There can be only one.
	return nil
}

// adoptExisting gives the Annotation the identity of the stored record for the same User and URL,
// so that the next save updates that record instead of inserting a second one.
func (service *Annotation) adoptExisting(session data.Session, annotation *model.Annotation) error {

	const location = "service.Annotation.adoptExisting"

	existing := model.NewAnnotation()

	if err := service.LoadByURL(session, annotation.UserID, annotation.URL, &existing); err != nil {
		return derp.Wrap(err, location, "Loading existing Annotation", annotation.URL)
	}

	annotation.AnnotationID = existing.AnnotationID
	annotation.Journal = existing.Journal

	return nil
}

// Delete removes an Annotation from the database (virtual delete)
func (service *Annotation) Delete(session data.Session, annotation *model.Annotation, note string) error {

	const location = "service.Annotation.Delete"

	if err := service.collection(session).Delete(annotation, note); err != nil {
		return derp.Wrap(err, location, "Deleting Annotation", annotation, note)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Annotation) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Annotation record, without applying any additional business rules
func (service *Annotation) HardDeleteByID(session data.Session, userID primitive.ObjectID, annotationID primitive.ObjectID) error {

	const location = "service.Annotation.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", annotationID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Annotation", "userID: "+userID.Hex(), "annotationID: "+annotationID.Hex())
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Annotation) ObjectType() string {
	return "Annotation"
}

// ObjectNew returns a fully initialized model.Annotation as a data.Object.
func (service *Annotation) ObjectNew() data.Object {
	result := model.NewAnnotation()
	return &result
}

// ObjectID returns the unique ID of the provided Annotation. Implements the ModelService interface.
func (service *Annotation) ObjectID(object data.Object) primitive.ObjectID {

	if annotation, ok := object.(*model.Annotation); ok {
		return annotation.AnnotationID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Annotation that matches the provided criteria. Implements the ModelService interface.
func (service *Annotation) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Annotation as a data.Object. Implements the ModelService interface.
func (service *Annotation) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewAnnotation()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Annotation in the database. Implements the ModelService interface.
func (service *Annotation) ObjectSave(session data.Session, object data.Object, comment string) error {
	if annotation, ok := object.(*model.Annotation); ok {
		return service.Save(session, annotation, comment)
	}
	return derp.Internal("service.Annotation.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a Annotation as deleted. Implements the ModelService interface.
func (service *Annotation) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if annotation, ok := object.(*model.Annotation); ok {
		return service.Delete(session, annotation, comment)
	}
	return derp.Internal("service.Annotation.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Annotation. Implements the ModelService interface.
func (service *Annotation) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Annotation", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Annotation
func (service *Annotation) Schema() schema.Schema {
	return schema.New(model.AnnotationSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser returns every Annotation that belongs to the provided User
func (service *Annotation) QueryByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) ([]model.Annotation, error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.Validation("UserID cannot be zero")
	}

	// Query the database
	criteria := exp.Equal("userId", userID)
	return service.Query(session, criteria, options...)
}

// LoadByID loads a single model.Annotation object that matches the provided annotationID
func (service *Annotation) LoadByID(session data.Session, userID primitive.ObjectID, annotationID primitive.ObjectID, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: Require a valid AnnotationID
	if annotationID.IsZero() {
		return derp.Validation("AnnotationID cannot be zero")
	}

	// Query the database
	criteria := exp.Equal("_id", annotationID).AndEqual("userId", userID)
	return service.Load(session, criteria, result)
}

// LoadByToken retrieves a single Annotation using its URL token
func (service *Annotation) LoadByToken(session data.Session, userID primitive.ObjectID, token string, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: Require a valid Token
	if token == "" {
		return derp.Validation("Token cannot be empty")
	}

	if annotationID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(session, userID, annotationID, result)
	}

	return derp.Validation("Token must be a valid ObjectID", token)
}

// LoadByURL loads the single model.Annotation that this User made against the provided URL
func (service *Annotation) LoadByURL(session data.Session, userID primitive.ObjectID, url string, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: Require a valid URL
	if url == "" {
		return derp.Validation("URL cannot be empty")
	}

	// Query the database
	criteria := exp.Equal("userId", userID).AndEqual("url", url)
	return service.Load(session, criteria, result)
}

// RangeByUserID returns a RangeFunc that yields all Annotations owned by the provided UserID
func (service *Annotation) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Annotation], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// DeleteByUserID deletes all Annotations owned by the provided UserID
func (service *Annotation) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Annotation.DeleteByUserID"

	// Retrieve all Annotations
	annotations, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Querying annotations by UserID", userID)
	}

	// Delete each annotation
	for annotation := range annotations {
		if err := service.Delete(session, &annotation, note); err != nil {
			return derp.Wrap(err, location, "Deleting Annotation", annotation)
		}
	}

	// Success
	return nil
}
