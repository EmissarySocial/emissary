package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/ghost/service"
)

// StepTopLevelDelete represents an action that can delete a top-level folder from the Domain
type StepTopLevelDelete struct {
	streamService *service.Stream
	parent        string
	templateID    string
}

// NewStepTopLevelDelete returns a fully parsed StepTopLevelDelete object
func NewStepTopLevelDelete(streamService *service.Stream, config datatype.Map) StepTopLevelDelete {

	return StepTopLevelDelete{
		streamService: streamService,
		parent:        config.GetString("parent"),
		templateID:    config.GetString("templateId"),
	}
}

func (step StepTopLevelDelete) Get(buffer io.Writer, renderer *Stream) error {
	return nil
}

func (step StepTopLevelDelete) Post(buffer io.Writer, renderer *Stream) error {
	return nil
}
