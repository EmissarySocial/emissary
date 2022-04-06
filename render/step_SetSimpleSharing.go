package render

import (
	"io"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"github.com/whisperverse/whisperverse/model"
)

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

func (step StepSetSimpleSharing) Get(renderer Renderer, buffer io.Writer) error {

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	model := streamRenderer.stream.Criteria.SimpleModel()

	// Try to write form HTML
	schema := step.schema()
	form := step.form()

	formHTML, err := form.HTML(factory.FormLibrary(), &schema, model)

	if err != nil {
		return derp.Wrap(err, "render.StepSetSimpleSharing.Get", "Error rendering form")
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().InnerHTML(step.Title).Close()
	b.Div().Class("space-below").InnerHTML(step.Message).Close()

	// Form
	b.Form("", "").
		Data("hx-post", renderer.URL()).
		Data("hx-swap", "none").
		Data("hx-push-url", "false").
		Script("init send checkFormRules(changed:me as Values)").
		EndBracket()

	b.WriteString(formHTML)
	b.Div()
	b.Button().Type("submit").Class("primary").InnerHTML("Save Changes").Close()
	b.Button().Type("button").Script("on click trigger closeModal").InnerHTML("Cancel").Close()
	b.CloseAll()

	// Write it to the output buffer and quit
	io.WriteString(buffer, b.String())
	return nil
}

func (step StepSetSimpleSharing) Post(renderer Renderer, buffer io.Writer) error {

	streamRenderer := renderer.(*Stream)
	request := streamRenderer.context().Request()

	// Try to parse the form input
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "render.StepSetSimpleSharing", "Error parsing form input")
	}

	stream := streamRenderer.stream
	stream.Criteria = model.NewCriteria()

	var groupIDs []string

	// If PUBLIC is checked, then roles are given to the public group.
	if convert.Bool(request.Form["public"]) {
		stream.Criteria.Public = step.Roles
		return nil
	}

	// Fall through means that roles are given to the selected groupIDs
	groupIDs = request.Form["groupIds"]

	for _, groupID := range groupIDs {
		stream.Criteria.Groups[groupID] = step.Roles
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
			{Kind: "select", Path: "groupIds", Options: form.Map{"provider": "groups"}, Show: form.Rule{Path: "public", Value: "'false'"}},
		},
	}
}
