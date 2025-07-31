package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FolderLookupProvider struct {
	session       data.Session
	folderService *Folder
	userID        primitive.ObjectID
}

func NewFolderLookupProvider(session data.Session, folderService *Folder, userID primitive.ObjectID) FolderLookupProvider {
	return FolderLookupProvider{
		session:       session,
		folderService: folderService,
		userID:        userID,
	}
}

func (service FolderLookupProvider) Group(name string) form.LookupGroup {
	return service
}

func (service FolderLookupProvider) Get() []form.LookupCode {

	if service.userID.IsZero() {
		return make([]form.LookupCode, 0)
	}

	folders, _ := service.folderService.QueryByUserID(service.session, service.userID)
	result := make([]form.LookupCode, 0, len(folders))

	for _, folder := range folders {
		result = append(result, folder.LookupCode())
	}

	return result
}

func (service FolderLookupProvider) Add(name string) (string, error) {

	const location = "service.FolderLookupProvider.Add"

	// RULE: Must have a valid UserID to add a Folder
	if service.userID.IsZero() {
		return "", derp.InternalError(location, "Cannot add folder to anonymous user")
	}

	folder := model.NewFolder()

	// RULE: Search for existing folder with the same name
	err := service.folderService.LoadByLabel(service.session, service.userID, name, &folder)

	if err == nil {
		return folder.ID(), nil
	}

	if derp.IsNotFound(err) {

		folder.Label = name
		folder.UserID = service.userID

		if err := service.folderService.Save(service.session, &folder, "created"); err != nil {
			return "", derp.Wrap(err, location, "Error saving folder", name)
		}

		return folder.ID(), nil
	}

	return "", derp.Wrap(err, location, "Error searching for existing folder", name)
}
