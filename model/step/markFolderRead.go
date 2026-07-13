package step

import (
	"github.com/benpate/rosetta/mapof"
)

// MarkFolderRead is a Step that marks every unread NewsItem in a folder (identified by the
// "folderId" query parameter) as read.  Used when the folder is opened (Mastodon-style).
type MarkFolderRead struct{}

// NewMarkFolderRead returns a fully initialized MarkFolderRead object
func NewMarkFolderRead(stepInfo mapof.Any) (MarkFolderRead, error) {
	return MarkFolderRead{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step MarkFolderRead) Name() string {
	return "mark-folder-read"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step MarkFolderRead) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step MarkFolderRead) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step MarkFolderRead) RequiredRoles() []string {
	return []string{}
}
