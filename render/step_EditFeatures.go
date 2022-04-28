package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

type StepEditFeatures struct{}

type stepEditFeaturesTransaction struct {
	TemplateIDs []string `path:"templateIds" form:"templateIds"`
}

func (step StepEditFeatures) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditFeatures.Get"

	factory := renderer.factory()
	streamService := factory.Stream()
	formLibrary := factory.FormLibrary()

	features, selected, err := streamService.ListAllFeaturesBySelectionAndRank(renderer.objectID())

	if err != nil {
		return derp.Wrap(err, location, "Error getting features")
	}

	s := schema.Schema{
		Element: schema.Array{Items: schema.String{}},
	}

	f := form.Form{
		Kind:  "layout-vertical",
		Label: "Select Features",
		Children: []form.Form{
			{Kind: "multiselect", Path: "templateIds", Options: form.Map{"options": features}},
		},
	}

	v := stepEditFeaturesTransaction{
		TemplateIDs: selected,
	}

	// Generate the form HTML
	html, err := f.HTML(formLibrary, &s, v)

	if err != nil {
		return derp.Wrap(err, location, "Error generating Form")
	}

	// Wrap it up and ship it out
	html = WrapForm(renderer.URL(), html)
	buffer.Write([]byte(html))

	return nil
}

func (step StepEditFeatures) Post(renderer Renderer) error {

	const location = "render.StepEditFeatures.Post"

	stream := renderer.(*Stream).stream

	streamService := renderer.factory().Stream()
	transaction := stepEditFeaturesTransaction{}

	// Try to collect transaction data from the form POST
	if err := renderer.context().Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Error binding transaction")
	}

	// For each selected template, guarantee that it is now listed as a feature in the correct order.
	if err := streamService.CreateAndSortFeatures(stream, transaction.TemplateIDs); err != nil {
		return derp.Wrap(err, location, "Error creating/restoring templates", transaction)
	}

	// Try to delete unused features
	if err := streamService.DeleteUnusedFeatures(stream.StreamID, transaction.TemplateIDs); err != nil {
		return derp.Wrap(err, location, "Error deleting unused features", transaction)
	}

	// Celebreate good times, come on!
	return nil
}

func (step StepEditFeatures) UseGlobalWrapper() bool {
	return true
}
