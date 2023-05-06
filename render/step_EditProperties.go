package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// StepEditProperties represents an action-step that can edit/update Container in a streamDraft.
type StepEditProperties struct {
	Title string
	Paths []string
}

func (step StepEditProperties) Get(renderer Renderer, buffer io.Writer) error {

	schema := renderer.schema()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	element := form.Element{
		Type:     "layout-vertical",
		Label:    step.Title,
		Children: []form.Element{},
	}

	for _, path := range step.Paths {

		switch path {

		case "token":
			element.Children = append(element.Children,
				form.Element{
					Path:        path,
					Type:        "text",
					Label:       "URL Token",
					Options:     mapof.Any{"format": "token"},
					Description: "Human-friendly web address",
				})

		case "label":
			element.Children = append(element.Children,
				form.Element{
					Path:        path,
					Type:        "text",
					Label:       "Label",
					Description: "Displayed on navigation, pages, and indexes",
					Options:     mapof.Any{"maxlength": 100},
				})

		case "description":

			element.Children = append(element.Children,
				form.Element{
					Type:        "textarea",
					Path:        path,
					Label:       "Text Description",
					Description: "Long description displays on pages and indexes",
					Options:     mapof.Any{"maxlength": 1000},
				})

		}
	}

	// Create HTML for the form
	html, err := form.Editor(schema, element, stream, renderer.lookupProvider())

	if err != nil {
		return derp.Wrap(err, "render.StepEditProperties.Get", "Error generating form HTML")
	}

	result := WrapModalForm(renderer.context().Response(), renderer.URL(), html)
	_, err = buffer.Write([]byte(result))

	return err
}

func (step StepEditProperties) UseGlobalWrapper() bool {
	return true
}

func (step StepEditProperties) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepEditProperties.Post"
	context := renderer.context()
	body := mapof.NewAny()

	if err := context.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Error binding request body")
	}

	schema := renderer.schema()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	for _, path := range step.Paths {
		if value, ok := body[path]; ok {
			if err := schema.Set(stream, path, value); err != nil {
				return derp.Wrap(err, location, "Error setting value", path, body[path])
			}
		}
	}

	if err := schema.Validate(stream); err != nil {
		return derp.Wrap(err, location, "Error validating data", stream)
	}

	CloseModal(context, "")

	// Success!
	return nil
}
