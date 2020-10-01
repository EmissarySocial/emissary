package render

import (
	"github.com/benpate/ghost/model"
)

// FolderListItem renderer can
type FolderListItem struct {
	folder model.Folder
}

// NewFolderList converts a slice of model.Folder into a slice of FolderListItem
func NewFolderList(folders []model.Folder) []FolderListItem {

	result := []FolderListItem{}

	for index := range folders {
		result = append(result, NewFolderListItem(folders[index]))
	}

	return result
}

// NewFolderListItem returns a fully initialized FolderListItem renderer
func NewFolderListItem(folder model.Folder) FolderListItem {

	return FolderListItem{
		folder: folder,
	}
}

// SubFolders returns a slice of all sub-folders within this Folder.
func (w FolderListItem) SubFolders() []FolderListItem {
	return NewFolderList(w.folder.SubFolders)
}

// FolderID returns the unique identifier of this folder.
func (w FolderListItem) Token() string {
	return w.folder.Token
}

// Label returns the human-friendly label for this folder.
func (w FolderListItem) Label() string {
	return w.folder.Label
}
