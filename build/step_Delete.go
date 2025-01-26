package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepDelete is a Step that can delete a Stream from the Domain
type StepDelete struct {
	Title   *template.Template
	Message *template.Template
	Submit  string
	Method  string
}

// Get displays a customizable confirmation form for the delete
func (step StepDelete) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Method == "post" {
		return Continue()
	}

	b := html.New()

	b.H1().InnerText(executeTemplate(step.Title, builder)).Close()
	b.Div().Class("margin-bottom").InnerText(executeTemplate(step.Message, builder)).Close()

	b.Button().Class("warning").
		Attr("hx-post", builder.URL()).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		InnerText(step.Submit).
		Close()

	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()
	b.CloseAll()

	modalHTML := WrapModal(builder.response(), b.String())

	// nolint:errcheck
	io.WriteString(buffer, modalHTML)

	return Halt().AsFullPage()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDelete) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepDelete.Post"

	if step.Method == "get" {
		return Continue()
	}

	// Delete the object via the model service.
	if err := builder.service().ObjectDelete(builder.object(), "Deleted"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error deleting stream"))
	}

	// If this object is also a SearchResulter, then we're gonna remove it from the search index
	if searchResult := getSearchResult(builder); searchResult.URL != "" {

		searchResultService := builder.factory().Search()

		// Delete step here
		if err := searchResultService.Delete(&searchResult, "unpublished"); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error deleting search result", searchResult))
		}
	}

	return Continue().WithEvent("closeModal", "true")
}
