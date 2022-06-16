package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
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

	factory := renderer.factory()
	formLibrary := factory.FormLibrary()
	element := form.NewForm("layout-vertical")
	element.Label = step.Title

	for _, path := range step.Paths {

		switch path {

		case "token":
			element.Children = append(element.Children,
				form.Form{
					Path:        path,
					Kind:        "text",
					Label:       "URL Token",
					Options:     datatype.Map{"format": "token"},
					Description: "Human-friendly web address",
				})

		case "label":
			element.Children = append(element.Children,
				form.Form{
					Path:        path,
					Kind:        "text",
					Label:       "Label",
					Description: "Displayed on navigation, pages, and indexes",
					Options:     datatype.Map{"maxlength": 100},
				})

		case "description":

			element.Children = append(element.Children,
				form.Form{
					Kind:        "textarea",
					Path:        path,
					Label:       "Text Description",
					Description: "Long description displays on pages and indexes",
					Options:     datatype.Map{"maxlength": 1000},
				})

		}
	}

	html, err := element.HTML(formLibrary, &schema, stream)

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

func (step StepEditProperties) Post(renderer Renderer) error {

	const location = "render.StepEditProperties.Post"
	body := datatype.Map{}
	context := renderer.context()

	if err := context.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Error binding request body")
	}

	schema := renderer.schema()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	for _, path := range step.Paths {
		if err := schema.Set(&stream, path, body[path]); err != nil {
			return derp.Wrap(err, location, "Error setting value", path, body[path])
		}
	}

	if err := schema.Validate(stream); err != nil {
		return derp.Wrap(err, location, "Error validating data", stream)
	}

	CloseModal(context, "")

	// Success!
	return nil
}
