package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

type StepEditFeatures struct{}

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
			{Kind: "multiselect", Path: "templateId", Options: form.Map{"options": features}},
		},
	}

	v := map[string]any{"templateId": selected}

	html, err := f.HTML(formLibrary, &s, v)

	if err != nil {
		return derp.Wrap(err, location, "Error generating Form")
	}

	html = WrapForm(renderer.context().Path(), html)
	buffer.Write([]byte(html))

	return nil
}

func (step StepEditFeatures) UseGlobalWrapper() bool {
	return true
}

func (step StepEditFeatures) Post(renderer Renderer) error {
	return nil
}
