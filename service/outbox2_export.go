package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportCollection returns the IDs of every activity in a User's outbox, oldest first, for a data export
func (service *Outbox2) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {

	const location = "service.Outbox2.ExportCollection"

	criteria := exp.Equal("actorType", model.ActorTypeUser).AndEqual("actorId", userID)
	result := make([]model.IDOnly, 0)

	if err := service.collection(session).Query(&result, notDeleted(criteria), option.Fields("_id"), option.SortAsc("createDate")); err != nil {
		return nil, derp.Wrap(err, location, "Querying outbox activities", "userId: "+userID.Hex())
	}

	return result, nil
}

// ExportDocument returns one activity from a User's outbox as a JSON string, for a data export
func (service *Outbox2) ExportDocument(session data.Session, userID primitive.ObjectID, activityID primitive.ObjectID) (string, error) {

	const location = "service.Outbox2.ExportDocument"

	// Load the activity, scoped to this User's outbox
	item := model.NewOutboxItem()

	if err := service.LoadByID(session, model.ActorTypeUser, userID, activityID, &item); err != nil {
		return "", derp.Wrap(err, location, "Loading outbox activity", "activityId: "+activityID.Hex())
	}

	// The export keeps the blind recipients and the journal, because it goes only to the activity's owner
	result, err := json.Marshal(item)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling outbox activity", "activityId: "+activityID.Hex())
	}

	return string(result), nil
}
