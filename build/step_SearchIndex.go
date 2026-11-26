package build

import (
	"io"
	"text/template"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepSearchIndex is a Step that adds the object to the search index, or removes it
type StepSearchIndex struct {
	If *template.Template
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSearchIndex) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post syncs the object's SearchResult into the search index, or deletes it when `If` is false
func (step StepSearchIndex) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSearchIndex.Post"

	searchResultService := builder.factory().SearchResult()
	searchResult := getSearchResult(builder)

	// If the step.If is FALSE, then delete the searchResult no matter what
	if !convert.Bool(executeTemplate(step.If, builder)) {
		searchResult.DeleteDate = time.Now().Unix()
	}

	// Add/Update/Delete the searchResult here
	if err := searchResultService.Sync(builder.session(), searchResult); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving search result", searchResult))
	}

	return Continue()
}
