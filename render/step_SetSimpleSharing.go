package render

import (
	"io"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/schema"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (step StepSetSimpleSharing) UseGlobalWrapper() bool {
	return true
}

func (step StepSetSimpleSharing) Post(renderer Renderer) error {

	const location = "render.StepSetSimpleSharing.Post"

	request := renderer.context().Request()

	// Try to parse the form input
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "render.StepSetSimpleSharing", "Error parsing form input")
	}

	var groupIDs []string

	rule := convert.String(request.Form["rule"])

	switch rule {
	case "public":
		groupIDs = []string{model.MagicGroupIDAnonymous.Hex()}

	case "authenticated":
		groupIDs = []string{model.MagicGroupIDAuthenticated.Hex()}

	case "private":
		groupIDs = request.Form["groupIds"]

	default:
		return derp.NewBadRequestError(location, "Invalid rule: ", rule)
	}

	// Build the stream criteria
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream
	stream.Criteria = model.NewCriteria()

	for _, groupID := range groupIDs {
		if _, err := primitive.ObjectIDFromHex(groupID); err == nil {
			stream.Criteria.Groups[groupID] = step.Roles
		}
	}

	// Success!
	return nil
}

// schema returns the validating schema for this form
func (step StepSetSimpleSharing) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"rule":     schema.String{Default: "public"},
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
			{Kind: "select", Path: "rule", Options: form.Map{"format": "radio", "provider": "sharing"}},
			{Kind: "select", Path: "groupIds", Options: form.Map{"provider": "groups"}, Show: form.Rule{Path: "rule", Value: "'private'"}},
		},
	}
}
