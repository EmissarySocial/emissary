package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Rule to the new profile.
func (service *Rule) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Rule.Import"

	// Unmarshal the JSON document into a new Rule
	rule := model.NewRule()
	if err := json.Unmarshal(document, &rule); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = rule.RuleID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Rule into the new, local Rule
	rule.RuleID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &rule.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", RuleID: "+rule.RuleID.Hex()))
	}

	// Map the FollowingID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &rule.FollowingID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FollowingID", "UserID: "+user.UserID.Hex()+", RuleID: "+rule.RuleID.Hex()))
	}

	// Save the Rule to the database
	if err := service.Save(session, &rule, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Rule")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Rule from the database
func (service *Rule) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Rule.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
