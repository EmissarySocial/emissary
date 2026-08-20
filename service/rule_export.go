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

// ExportCollection returns the IDs of every Rule to include in a User's data export
func (service *Rule) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single Rule as a JSON string, for a User's data export
func (service *Rule) ExportDocument(session data.Session, userID primitive.ObjectID, ruleID primitive.ObjectID) (string, error) {

	const location = "service.Rule.ExportDocument"

	// Load the Rule
	rule := model.NewRule()
	if err := service.LoadByID(session, userID, ruleID, &rule); err != nil {
		return "", derp.Wrap(err, location, "Loading Rule")
	}

	// Marshal the rule as JSON
	result, err := json.Marshal(rule)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling Rule", rule)
	}

	// Success
	return string(result), nil
}
