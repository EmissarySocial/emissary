package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/ghost/service"
)

// StepTopFolderDelete represents an action that can delete a top-level folder from the Domain
type StepTopFolderDelete struct {
	streamService *service.Stream
	parent        string
	templateID    string
}

// NewStepTopFolderDelete returns a fully parsed StepTopFolderDelete object
func NewStepTopFolderDelete(streamService *service.Stream, config datatype.Map) StepTopFolderDelete {

	return StepTopFolderDelete{
		streamService: streamService,
		parent:        config.GetString("parent"),
		templateID:    config.GetString("templateId"),
	}
}

func (step StepTopFolderDelete) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepTopFolderDelete) Post(buffer io.Writer, renderer *Renderer) error {
	return nil
}
