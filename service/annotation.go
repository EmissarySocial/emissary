package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Annotation manages all interactions with the Annotation collection
type Annotation struct {
	collection      data.Collection
	activityService *ActivityStream
}

// NewAnnotation returns a fully populated Annotation service
func NewAnnotation() Annotation {
	return Annotation{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Annotation) Refresh(collection data.Collection, activityService *ActivityStream) {
	service.collection = collection
	service.activityService = activityService
}

// Close stops any background processes controlled by this service
func (service *Annotation) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Annotation) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Annotation) Query(criteria exp.Expression, options ...option.Option) ([]model.Annotation, error) {
	result := make([]model.Annotation, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Annotations who match the provided criteria
func (service *Annotation) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Annotation from the database
func (service *Annotation) Load(criteria exp.Expression, result *model.Annotation) error {

	const location = "service.Annotation.Load"

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Error loading Annotation", criteria)
	}

	return nil
}

// Save adds/updates an Annotation in the database
func (service *Annotation) Save(annotation *model.Annotation, note string) error {

	const location = "service.Annotation.Save"

	// Copy values from the annotated document
	document, err := service.activityService.Load(annotation.URL)

	if err != nil {
		return derp.Wrap(err, location, "Error loading annotated document", annotation.URL)
	}

	annotation.Name = document.Name()
	annotation.Icon = document.Icon().Href()

	// Validate the value before saving
	if err := service.Schema().Validate(annotation); err != nil {
		return derp.Wrap(err, location, "Error validating Annotation", annotation)
	}

	// Save the value to the database
	if err := service.collection.Save(annotation, note); err != nil {
		return derp.Wrap(err, location, "Error saving Annotation", annotation, note)
	}

	return nil
}

// Delete removes an Annotation from the database (virtual delete)
func (service *Annotation) Delete(annotation *model.Annotation, note string) error {

	const location = "service.Annotation.Delete"

	if err := service.collection.Delete(annotation, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Annotation", annotation, note)
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

// New returns a fully initialized model.Annotation as a data.Object.
func (service *Annotation) ObjectNew() data.Object {
	result := model.NewAnnotation()
	return &result
}

func (service *Annotation) ObjectID(object data.Object) primitive.ObjectID {

	if annotation, ok := object.(*model.Annotation); ok {
		return annotation.AnnotationID
	}

	return primitive.NilObjectID
}

func (service *Annotation) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Annotation) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewAnnotation()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Annotation) ObjectSave(object data.Object, comment string) error {
	if annotation, ok := object.(*model.Annotation); ok {
		return service.Save(annotation, comment)
	}
	return derp.InternalError("service.Annotation.ObjectSave", "Invalid Object Type", object)
}

func (service *Annotation) ObjectDelete(object data.Object, comment string) error {
	if annotation, ok := object.(*model.Annotation); ok {
		return service.Delete(annotation, comment)
	}
	return derp.InternalError("service.Annotation.ObjectDelete", "Invalid Object Type", object)
}

func (service *Annotation) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Annotation", "Not Authorized")
}

func (service *Annotation) Schema() schema.Schema {
	return schema.New(model.AnnotationSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Annotation) QueryByUser(userID primitive.ObjectID, options ...option.Option) ([]model.Annotation, error) {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	// Query the database
	criteria := exp.In("_id", userID)
	return service.Query(criteria, options...)
}

// LoadByID loads a single model.Annotation object that matches the provided annotationID
func (service *Annotation) LoadByID(userID primitive.ObjectID, annotationID primitive.ObjectID, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid AnnotationID
	if annotationID.IsZero() {
		return derp.ValidationError("AnnotationID cannot be zero")
	}

	// Query the database
	criteria := exp.Equal("_id", annotationID).AndEqual("userId", userID)
	return service.Load(criteria, result)
}

func (service *Annotation) LoadByToken(userID primitive.ObjectID, token string, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid Token
	if token == "" {
		return derp.ValidationError("Token cannot be empty")
	}

	if annotationID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(userID, annotationID, result)
	}

	return derp.ValidationError("Token is must be a valid ObjectID", token)
}

// LoadByID loads a single model.Annotation object that matches the provided annotationID
func (service *Annotation) LoadByURL(userID primitive.ObjectID, url string, result *model.Annotation) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid URL
	if url == "" {
		return derp.ValidationError("URL cannot be empty")
	}

	// Query the database
	criteria := exp.Equal("userId", userID).AndEqual("url", url)
	return service.Load(criteria, result)
}

/******************************************
 * Custom Behaviors
 ******************************************/
