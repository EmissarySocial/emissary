package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

// ImportSave is a part of the "Importable" interface, and saves an imported User to the new profile.
func (service *User) Import(session data.Session, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.User.Import"

	importedUser := model.NewUser()

	// Unmarshal the document into the new User
	if err := json.Unmarshal(document, &importedUser); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	importItem.RemoteID = importedUser.UserID
	importItem.LocalID = user.UserID

	/*
		// Map values from the original User into the new, local User?
		user.DisplayName = importedUser.DisplayName
		user.StatusMessage = importedUser.StatusMessage
		user.Location = importedUser.Location
		user.Locale = importedUser.Locale
		user.Hashtags = importedUser.Hashtags
		user.Links = importedUser.Links
		user.Data = importedUser.Data
		user.IsPublic = importedUser.IsPublic
		user.IsIndexable = importedUser.IsIndexable

		// Save the User to the database
		if err := service.Save(session, user, "Imported"); err != nil {
			return derp.Wrap(err, location, "Unable to save imported User")
		}
	*/

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported User from the database
func (service *User) UndoImport(session data.Session, importItem *model.ImportItem) error {

	// We can't DELETE or UNDO any changes made to the User.
	return nil
}
