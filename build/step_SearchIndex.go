package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepSearchIndex is a Step that can update a stream's PublishDate with the current time.
type StepSearchIndex struct {
	Action string
	If     *template.Template
}

func (step StepSearchIndex) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSearchIndex) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSearchIndex.Post"

	// Check the "IF" condition to see if this step should run...
	if !convert.Bool(executeTemplate(step.If, builder)) {
		return nil
	}

	if searchResult, ok := getSearchResult(builder); ok {

		searchResultService := builder.factory().Search()

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
	}

	return nil
}
