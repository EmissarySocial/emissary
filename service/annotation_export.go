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

func (service *Annotation) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Annotation) ExportDocument(session data.Session, userID primitive.ObjectID, annotationID primitive.ObjectID) (string, error) {

	const location = "service.Annotation.ExportDocument"

	// Load the Annotation
	annotation := model.NewAnnotation()
	if err := service.LoadByID(session, userID, annotationID, &annotation); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Annotation")
	}

	// Marshal the annotation as JSON
	result, err := json.Marshal(annotation)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Annotation", annotation)
	}

	// Success
	return string(result), nil
}
