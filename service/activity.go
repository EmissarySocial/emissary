package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/whisperverse/whisperverse/model"
)

// Activity manages all interactions with the Activity collection
type Activity struct {
	collection data.Collection
}

// NewActivity returns a fully populated Activity service
func NewActivity(collection data.Collection) Activity {
	return Activity{
		collection: collection,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Activitys who match the provided criteria
func (service *Activity) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Activity from the database
func (service *Activity) Load(criteria exp.Expression, result *model.Activity) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Activity", "Error loading Activity", criteria)
	}

	return nil
}

// Save adds/updates an Activity in the database
func (service *Activity) Save(activity *model.Activity, note string) error {

	// First, hard delete any other activities on this stream
	criteria := exp.Equal("userId", activity.UserID).AndEqual("streamId", activity.StreamID)
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Activity", "Error deleting previous Activity", criteria)
	}

	// Now we can save the new Activity without duplicates
	if err := service.collection.Save(activity, note); err != nil {
		return derp.Wrap(err, "service.Activity", "Error saving Activity", activity, note)
	}

	return nil
}

// Delete removes an Activity from the database (virtual delete)
func (service *Activity) Delete(activity *model.Activity, note string) error {

	if err := service.collection.Delete(activity, note); err != nil {
		return derp.Wrap(err, "service.Activity", "Error deleting Activity", activity, note)
	}

	return nil
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

/*******************************************
 * CUSTOM ACTIONS
 *******************************************/

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Activity as a data.Object.
func (service *Activity) ObjectNew() data.Object {
	result := model.NewActivity()
	return &result
}

func (service *Activity) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Activity) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewActivity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Activity) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.Activity), comment)
}

func (service *Activity) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.Activity), comment)
}

func (service *Activity) Debug() datatype.Map {
	return datatype.Map{
		"service": "Activity",
	}
}
