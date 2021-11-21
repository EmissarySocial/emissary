package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/ghost/service"
)

// StepTopFolderEdit represents an action that can edit a top-level folder in the Domain
type StepTopFolderEdit struct {
	streamService *service.Stream
	parent        string
	templateID    string
}

// NewStepTopFolderEdit returns a fully parsed StepTopFolderEdit object
func NewStepTopFolderEdit(streamService *service.Stream, config datatype.Map) StepTopFolderEdit {

	return StepTopFolderEdit{
		streamService: streamService,
		parent:        config.GetString("parent"),
		templateID:    config.GetString("templateId"),
	}
}

func (step StepTopFolderEdit) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepTopFolderEdit) Post(buffer io.Writer, renderer *Renderer) error {
	return nil
}
