package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FolderLookupProvider struct {
	folderService *Folder
	userID        primitive.ObjectID
}

func NewFolderLookupProvider(folderService *Folder, userID primitive.ObjectID) FolderLookupProvider {
	return FolderLookupProvider{
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

	folders, _ := service.folderService.QueryByUserID(service.userID)
	result := make([]form.LookupCode, 0, len(folders))

	for _, folder := range folders {
		result = append(result, folder.LookupCode())
	}

	return result
}

func (service FolderLookupProvider) Add(name string) (string, error) {

	// RULE: Must have a valid UserID to add a Folder
	if service.userID.IsZero() {
		return "", derp.InternalError("service.FolderLookupProvider.Add", "Cannot add folder to anonymous user")
	}

	folder := model.NewFolder()

	// RULE: Search for existing folder with the same name
	err := service.folderService.LoadByLabel(service.userID, name, &folder)

	if err == nil {
		return folder.ID(), nil
	}

	if derp.IsNotFound(err) {

		folder.Label = name
		folder.UserID = service.userID

		if err := service.folderService.Save(&folder, "created"); err != nil {
			return "", derp.Wrap(err, "service.FolderLookupProvider.Add", "Error saving folder", name)
		}

		return folder.ID(), nil
	}

	return "", derp.Wrap(err, "service.FolderLookupProvider.Add", "Error searching for existing folder", name)
}
