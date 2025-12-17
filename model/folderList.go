package model

import (
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FolderList contains a group of folders and the currently selected folder.
// This is used by the Inbox templates to display all folders for a User.
type FolderList struct {
	Folders    sliceof.Object[Folder] `json:"folders"`
	Section    string                 `json:"section"`
	SelectedID primitive.ObjectID     `json:"selectedId"`
}

// NewFolderList returns a fully initialized FolderList object
func NewFolderList() FolderList {
	return FolderList{
		Folders: sliceof.NewObject[Folder](),
	}
}

// Selected returns the currently selected folder
func (list FolderList) Selected() Folder {

	// Try to find the folder in the available list
	if !list.SelectedID.IsZero() {
		for _, folder := range list.Folders {
			if folder.FolderID == list.SelectedID {
				return folder
			}
		}
	}

	// Otherwise, make a synthetic "All Folders" result.
	all := NewFolder()
	all.FolderID = primitive.NilObjectID
	all.Label = "All Folders"
	all.Icon = "folder"
	all.Layout = "SOCIAL"
	return all
}

// HasSelection returns TRUE if a folder is currently selected
func (list FolderList) HasSelection() bool {
	return list.SelectedID != primitive.NilObjectID
}

// ByID scans the list for the folder with the specified ID
func (list FolderList) ByID(folderID primitive.ObjectID) Folder {
	for _, folder := range list.Folders {
		if folder.FolderID == folderID {
			return folder
		}
	}

	return NewFolder()
}
