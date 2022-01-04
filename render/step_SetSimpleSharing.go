package render

import (
	"io"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
	"github.com/benpate/null"
	"github.com/benpate/schema"
)

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	formLibrary *form.Library
	title       string
	message     string
	roles       []string
}

// NewStepSetSimpleSharing returns a fully parsed StepSetSimpleSharing object
func NewStepSetSimpleSharing(formLibrary *form.Library, stepInfo datatype.Map) StepSetSimpleSharing {

	return StepSetSimpleSharing{
		formLibrary: formLibrary,
		title:       stepInfo.GetString("title"),
		message:     stepInfo.GetString("message"),
		roles:       stepInfo.GetSliceOfString("roles"),
	}
}

func (step StepSetSimpleSharing) Get(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(*Stream)
	model := streamRenderer.stream.Criteria.SimpleModel()

	// Try to write form HTML
	schema := step.schema()
	form := step.form()

	formHTML, err := form.HTML(step.formLibrary, &schema, model)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepSetSimpleSharing.Get", "Error rendering form")
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()
	b.Container("HR").Close()

	// Form
	b.Form("", "").Data("hx-post", renderer.URL()).Data("hx-push-url", "false").Data("hx-swap", "none").EndBracket()
	b.WriteString(formHTML)
	b.Div()
	b.Button().Type("submit").Class("primary").InnerHTML("Save Changes").Close()
	b.Button().Type("button").Script("on click send closeModal to #modal").InnerHTML("Cancel").Close()
	b.CloseAll()

	// Write it to the output buffer and quit
	io.WriteString(buffer, b.String())
	return nil
}

func (step StepSetSimpleSharing) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(*Stream)
	request := streamRenderer.context().Request()

	// Try to parse the form input
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "ghost.render.StepSetSimpleSharing", "Error parsing form input")
	}

	stream := streamRenderer.stream
	stream.Criteria = model.NewCriteria()

	var groupIDs []string

	// If PUBLIC is checked, then roles are given to the public group.
	if convert.Bool(request.Form["public"]) {
		stream.Criteria.Public = step.roles
		return nil
	}

	// Fall through means that roles are given to the selected groupIDs
	groupIDs = request.Form["groupIds"]

	for _, groupID := range groupIDs {
		stream.Criteria.Groups[groupID] = step.roles
	}

	return nil
}

// schema returns the validating schema for this form
func (step StepSetSimpleSharing) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"public":   schema.Boolean{Default: null.NewBool(true)},
				"groupIds": schema.Array{Items: schema.String{Format: "objectId"}},
			},
		},
	}
}

// form returns the form to be displayed
func (step StepSetSimpleSharing) form() form.Form {

	return form.Form{
		Kind: "layout-vertical",
		Children: []form.Form{
			{Kind: "select", Path: "public", Options: form.Map{"format": "radio", "provider": "sharing"}},
			{Kind: "select", Path: "groupIds", Options: form.Map{"provider": "groups"}},
		},
	}
}
