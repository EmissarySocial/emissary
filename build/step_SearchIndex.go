package build

import (
	"io"
	"text/template"
	"time"

	"github.com/EmissarySocial/emissary/model"
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

	searchResultService := builder.factory().SearchResult()
	searchResult := getSearchResult(builder)

	// If the step.If is FALSE, then delete the searchResult no matter what
	if !convert.Bool(executeTemplate(step.If, builder)) {
		searchResult.DeleteDate = time.Now().Unix()
	}

	// Add/Update/Delete the searchResult here
	if err := searchResultService.Sync(searchResult); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving search result", searchResult))
	}

	return Continue()
}

// nolint unused
func (step StepSearchIndex) isSearchable(builder Builder, searchResult model.SearchResult) bool {

	if searchResult.IsZero() {
		return false
	}

	if step.Action == "delete" {
		return false
	}

	return convert.Bool(executeTemplate(step.If, builder))

}
