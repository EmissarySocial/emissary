package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported NewsFeed to the new profile.
func (service *NewsFeed) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.NewsFeed.Import"

	// Unmarshal the JSON document into a new NewsFeed
	newsItem := model.NewNewsItem()
	if err := json.Unmarshal(document, &newsItem); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = newsItem.NewsItemID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original NewsFeed into the new, local NewsFeed
	newsItem.NewsItemID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &newsItem.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", NewsItemID: "+newsItem.NewsItemID.Hex()))
	}

	// Map the FollowingID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &newsItem.FollowingID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FollowingID", "UserID: "+user.UserID.Hex()+", NewsItemID: "+newsItem.NewsItemID.Hex()))
	}

	// Map the FolderID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &newsItem.FolderID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FolderID", "UserID: "+user.UserID.Hex()+", NewsItemID: "+newsItem.NewsItemID.Hex()))
	}

	// Save the NewsFeed to the database
	if err := service.Save(session, &newsItem, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported NewsFeed")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported NewsFeed from the database
func (service *NewsFeed) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.NewsFeed.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
