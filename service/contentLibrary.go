package service

import "github.com/benpate/ghost/content"

type ContentLibrary struct{}

// Viewer returns a content library used to VIEW content.Items
func (contentLibrary ContentLibrary) Viewer() content.Library {
	return content.ViewerLibrary()
}

// Viewer returns a content library used to EDIT content.Items
func (contentLibrary ContentLibrary) Editor() content.Library {
	return content.EditorLibrary()
}
