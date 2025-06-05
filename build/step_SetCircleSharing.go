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
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetCircleSharing represents an action that can edit a top-level folder in the Domain
type StepSetCircleSharing struct {
	Title   string
	Message string
	Roles   []string
}

func (step StepSetCircleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	streamBuilder := builder.(Stream)
	model := step.SimplePermissionModel(streamBuilder._stream)

	// Try to write form HTML
	formHTML, err := form.Editor(step.schema(), step.form(), model, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetCircleSharing.Get", "Error building form"))
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

func (step StepSetCircleSharing) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetCircleSharing.Post"

	request := builder.request()

	// Try to parse the form input
	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetCircleSharing", "Error parsing form input"))
	}

	var circleIDs []primitive.ObjectID

	rule := convert.String(request.Form["rule"])

	switch rule {
	case "anonymous":
		circleIDs = []primitive.ObjectID{model.MagicGroupIDAnonymous}

	case "authenticated":
		circleIDs = []primitive.ObjectID{model.MagicGroupIDAuthenticated}

	case "private":
		circleIDs = id.SliceOfID(request.Form["circleIds"])

	default:
		return Halt().WithError(derp.BadRequestError(location, "Invalid rule: ", rule))
	}

	// Build the stream criteria
	streamBuilder := builder.(Stream)
	stream := streamBuilder._stream
	stream.Permissions = model.NewStreamPermissions()

	for _, circleID := range circleIDs {
		for _, role := range step.Roles {
			stream.AssignPermission(role, circleID)
		}
	}

	// Success!
	return nil
}

// schema returns the validating schema for this form
func (step StepSetCircleSharing) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"rule":      schema.String{Default: "anonymous"},
				"circleIds": schema.Array{Items: schema.String{Format: "objectId"}},
			},
		},
	}
}

// form returns the form to be displayed
func (step StepSetCircleSharing) form() form.Element {

	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "radio", Path: "rule", Options: mapof.Any{"provider": "sharing"}},
			{Type: "multiselect", Path: "circleIds", Options: mapof.Any{"provider": "circles", "show-if": "rule is private"}}, // TODO: MEDIUM: Restore conditional rules to form elements.  This one was: Show: form.Rule{Path: "rule", Value: "'private'"}
		},
	}
}

// SimplePermissionModel returns a model object for displaying Simple Sharing.
func (step StepSetCircleSharing) SimplePermissionModel(stream *model.Stream) mapof.Any {

	// Special case if this is for EVERYBODY
	if _, ok := stream.Permissions[model.MagicGroupIDAnonymous.Hex()]; ok {
		return mapof.Any{
			"rule":      "anonymous",
			"circleIds": sliceof.NewString(),
		}
	}

	// Special case if this is for AUTHENTICATED
	if _, ok := stream.Permissions[model.MagicGroupIDAuthenticated.Hex()]; ok {
		return mapof.Any{
			"rule":      "authenticated",
			"circleIds": sliceof.NewString(),
		}
	}

	// Fall through means that additional circles are selected.
	// First, get all keys to the Groups map
	circleIDs := make(sliceof.String, len(stream.Permissions))
	index := 0

	for circleID := range stream.Permissions {
		circleIDs[index] = circleID
		index++
	}

	return mapof.Any{
		"rule":      "private",
		"circleIds": circleIDs,
	}
}
