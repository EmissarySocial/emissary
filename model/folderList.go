package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// FolderList contains a group of folders and the currently selected folder
type FolderList struct {
	Folders    []Folder `json:"folders"`
	SelectedID primitive.ObjectID
}

// NewFolderList returns a fully initialized FolderList object
func NewFolderList() FolderList {
	return FolderList{
		Folders: make([]Folder, 0),
	}
}

// Selected returns the currently selected folder
func (list FolderList) Selected() Folder {
	for _, folder := range list.Folders {
		if folder.FolderID == list.SelectedID {
			return folder
		}
	}

	return NewFolder()
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
