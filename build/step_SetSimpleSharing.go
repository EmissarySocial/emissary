package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

func (step StepSetSimpleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	streamBuilder := builder.(*Stream)
	model := streamBuilder._stream.SimplePermissionModel()

	// Try to write form HTML
	formHTML, err := form.Editor(step.schema(), step.form(), model, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSimpleSharing.Get", "Error building form"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H2().InnerText(step.Title).Close()
	b.H3().InnerText(step.Message).Close()

	// Form
	b.Form("", "").
		Data("hx-post", builder.URL()).
		Data("hx-swap", "none").
		Data("hx-push-url", "false").
		Script("init send checkFormRules(changed:me as Values)").
		EndBracket()

	b.WriteString(formHTML)
	b.Div()
	b.Button().Type("submit").Class("primary").InnerText("Save Changes").Close()
	b.Button().Type("button").Script("on click trigger closeModal").InnerText("Cancel").Close()
	b.CloseAll()

	// nolint:errcheck
	io.WriteString(buffer, b.String())
	return nil
}

func (step StepSetSimpleSharing) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetSimpleSharing.Post"

	request := builder.request()

	// Try to parse the form input
	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSimpleSharing", "Error parsing form input"))
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
		return Halt().WithError(derp.NewBadRequestError(location, "Invalid rule: ", rule))
	}

	// Build the stream criteria
	streamBuilder := builder.(*Stream)
	stream := streamBuilder._stream
	stream.Permissions = model.NewStreamPermissions()

	for _, groupID := range groupIDs {
		for _, role := range step.Roles {
			stream.AssignPermission(role, groupID)
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
			{Type: "radio", Path: "rule", Options: mapof.Any{"provider": "sharing"}},
			{Type: "multiselect", Path: "groupIds", Options: mapof.Any{"provider": "groups", "show-if": "rule is private"}}, // TODO: MEDIUM: Restore conditional rules to form elements.  This one was: Show: form.Rule{Path: "rule", Value: "'private'"}
		},
	}
}
