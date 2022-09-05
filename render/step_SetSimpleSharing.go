package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

func (step StepSetSimpleSharing) Get(renderer Renderer, buffer io.Writer) error {

	streamRenderer := renderer.(*Stream)
	model := streamRenderer.stream.Permissions.SimpleModel()

	// Try to write form HTML
	factory := renderer.factory()
	formHTML, err := form.Editor(step.schema(), step.form(), model, factory.LookupProvider())

	if err != nil {
		return derp.Wrap(err, "render.StepSetSimpleSharing.Get", "Error rendering form")
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H2().InnerHTML(step.Title).Close()
	b.H3().InnerHTML(step.Message).Close()

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

	var groupIDs []primitive.ObjectID

	rule := convert.String(request.Form["rule"])

	switch rule {
	case "anonymous":
		groupIDs = []primitive.ObjectID{model.MagicGroupIDAnonymous}

	case "authenticated":
		groupIDs = []primitive.ObjectID{model.MagicGroupIDAuthenticated}

	case "private":
		groupIDs = id.SliceOfID(request.Form["groupIds"])

	default:
		return derp.NewBadRequestError(location, "Invalid rule: ", rule)
	}

	// Build the stream criteria
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream
	stream.Permissions = model.NewPermissions()

	for _, groupID := range groupIDs {
		for _, role := range step.Roles {
			stream.Permissions.Assign(role, groupID)
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
				"rule":     schema.String{Default: "anonymous"},
				"groupIds": schema.Array{Items: schema.String{Format: "objectId"}},
			},
		},
	}
}

// form returns the form to be displayed
func (step StepSetSimpleSharing) form() form.Element {

	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "radio", Path: "rule", Options: maps.Map{"provider": "sharing"}},
			{Type: "multiselect", Path: "groupIds", Options: maps.Map{"provider": "groups"}}, // TODO: , Show: form.Rule{Path: "rule", Value: "'private'"}
		},
	}
}
