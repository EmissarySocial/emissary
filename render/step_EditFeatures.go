package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

type StepEditFeatures struct{}

func (step StepEditFeatures) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditFeatures.Get"

	factory := renderer.factory()
	streamService := factory.Stream()

	features, selected, err := streamService.ListAllFeaturesBySelectionAndRank(renderer.objectID())

	if err != nil {
		return derp.Wrap(err, location, "Error getting features")
	}

	s := schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"templateIds": schema.Array{Items: schema.String{}},
			},
		},
	}

	featuresElement := form.Element{
		Type:  "layout-vertical",
		Label: "Add/Remove Features of this Stream",
		Children: []form.Element{{
			Type:        "multiselect",
			Path:        "templateIds",
			Description: "Check the features you want to add, drag to rearrange.",
			Options:     maps.Map{"options": features, "sort": true}},
		},
	}

	v := stepEditFeaturesTransaction{
		TemplateIDs: selected,
	}

	// Generate the form HTML
	html, err := form.Editor(s, featuresElement, v, factory.LookupProvider())

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

type stepEditFeaturesTransaction struct {
	TemplateIDs sliceof.String `form:"templateIds"`
}

func (txn *stepEditFeaturesTransaction) GetObjectOK(name string) (any, bool) {
	if name == "templateIds" {
		return &txn.TemplateIDs, true
	}
	return nil, false
}
