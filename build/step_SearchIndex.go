package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepSearchIndex is an action-step that can update a stream's PublishDate with the current time.
type StepSearchIndex struct {
	If     *template.Template
	Action string
}

func (step StepSearchIndex) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSearchIndex) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSearchIndex.Post"

	// Verify that this object is a SearchResulter
	searchResulter, ok := builder.object().(SearchResulter)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "Object must be a SearchResulter"))
	}

	// Check the "IF" condition to see if this step should run...
	if !convert.Bool(executeTemplate(step.If, builder)) {
		return nil
	}

	searchResultService := builder.factory().Search()
	searchResult := searchResulter.SearchResult()

	// Delete step here
	if step.Action == "delete" {
		if err := searchResultService.Delete(&searchResult, "deleted"); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error deleting search result", searchResult))
		}

		return nil
	}

	// Add/Update step here
	if err := searchResultService.Upsert(searchResult); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving search result", searchResult))
	}

	return nil
}
