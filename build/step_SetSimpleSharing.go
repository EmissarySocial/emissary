package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	Title   string
	Message string
	Role    string
}

func (step StepSetSimpleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetSimpleSharing.Get"

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Builder is not a StreamBuilder"))
	}

	// Calculate the value object for this step
	value := step.calculateValue(streamBuilder._stream)
	schema := step.schema()
	form, err := step.form()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error building form for StepSetSimpleSharing"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	if authorization := builder.authorization(); authorization.DomainOwner {
		b.A("/admin/groups/").InnerHTML("Edit Groups &rarr;").Close()
	}

	// Heading
	b.H2().InnerText(step.Title).Close()

	if step.Message != "" {
		b.H3().InnerText(step.Message).Close()
	}

	if err := form.Edit(&schema, builder.lookupProvider(), value, b); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error rendering form for StepSetSimpleSharing"))
	}

	b.CloseAll()

	result := WrapForm(builder.URL(), b.String(), "application/x-www-form-urlencoded")

	// Write the result to the buffer
	if _, err := io.WriteString(buffer, result); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing HTML to buffer"))
	}

	return nil
}

func (step StepSetSimpleSharing) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetSimpleSharing.Post"

	// Guarantee that we have a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Builder is not a StreamBuilder"))
	}

	// Try to parse the form input
	request := streamBuilder.request()

	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSimpleSharing", "Error parsing form input"))
	}

	// Reset mapped privileges for the Stream
	stream := streamBuilder._stream

	result := id.NewSlice()

	for _, permission := range request.Form["groupIds"] {

		// Verify we have a valid CircleID, then add it to the list of allowed Circles
		groupID, err := primitive.ObjectIDFromHex(permission)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid CircleID in form input"))
		}

		if groupID == model.MagicGroupIDAnonymous {
			result = id.Slice{model.MagicGroupIDAnonymous}
			break
		}

		if groupID == model.MagicGroupIDAuthenticated {
			result = id.Slice{model.MagicGroupIDAuthenticated}
			break
		}

		if groupID == model.MagicGroupIDOwners {
			result = id.Slice{model.MagicGroupIDOwners}
			break
		}

		// Otherwise, add the CircleID to the list of allowed Circles
		result = append(result, groupID)
	}

	if result.IsEmpty() {
		result = id.Slice{model.MagicGroupIDOwners}
	}

	stream.Groups[step.Role] = result

	// Done!
	return nil
}

// schema returns the validating schema for this form
func (step StepSetSimpleSharing) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"groupIds": schema.Array{Items: schema.String{}},
			},
		},
	}
}

// form returns the form to be displayed
func (step StepSetSimpleSharing) form() (form.Element, error) {

	// Build the form element
	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{
				Type: "check-button-group",
				Path: "groupIds",
				Options: mapof.Any{
					"class": "simple-sharing simple-sharing-not-group",
					"enum": []form.LookupCode{
						{
							Value:       model.MagicGroupIDAnonymous.Hex(),
							Label:       "Share with Everyone",
							Description: "Publicly visible to everyone on the Internet, signed in or not.",
							Icon:        "globe",
						},
						{
							Value:       model.MagicGroupIDAuthenticated.Hex(),
							Label:       "Signed-In Users Only",
							Description: "Visible to all authenticated website users.",
							Icon:        "person-circle",
						},
						{
							Value:       model.MagicGroupIDOwners.Hex(),
							Label:       "Domain Owners",
							Description: "Only visible to domain owners.",
							Icon:        "lock",
						},
					},
					"script": "on click tell <.simple-sharing /> set your.checked to false end then set my.checked to true",
				},
			},
			{
				Type:  "check-button-group",
				Path:  "groupIds",
				Label: "These Groups Only",
				Options: mapof.Any{
					"class":    "simple-sharing",
					"provider": "groups",
					"script":   "on click tell <.simple-sharing-not-group /> set your.checked to false",
				},
			},
		},
	}, nil
}

func (step StepSetSimpleSharing) calculateValue(stream *model.Stream) mapof.Object[id.Slice] {

	if groupIds := stream.Groups[step.Role]; groupIds.NotEmpty() {
		return mapof.Object[id.Slice]{
			"groupIds": groupIds,
		}
	}

	return mapof.Object[id.Slice]{
		"groupIds": id.Slice{model.MagicGroupIDOwners},
	}
}

/*

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

// StepSetSimpleSharing represents an action that can edit a top-level folder in the Domain
type StepSetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

func (step StepSetSimpleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetSimpleSharing.Get"

	streamBuilder := builder.(Stream)
	model := step.SimplePermissionModel(streamBuilder._stream)

	// Try to write form HTML
	formHTML, err := form.Editor(step.schema(), step.form(), model, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error building form"))
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

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing form HTML to buffer"))
	}

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
		return Halt().WithError(derp.BadRequestError(location, "Invalid rule: ", rule))
	}

	// Build the stream criteria
	streamBuilder := builder.(Stream)
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

// SimplePermissionModel returns a model object for displaying Simple Sharing.
func (step StepSetSimpleSharing) SimplePermissionModel(stream *model.Stream) mapof.Any {

	// Special case if this is for EVERYBODY
	if _, ok := stream.Permissions[model.MagicGroupIDAnonymous.Hex()]; ok {
		return mapof.Any{
			"rule":     "anonymous",
			"groupIds": sliceof.NewString(),
		}
	}

	// Special case if this is for AUTHENTICATED
	if _, ok := stream.Permissions[model.MagicGroupIDAuthenticated.Hex()]; ok {
		return mapof.Any{
			"rule":     "authenticated",
			"groupIds": sliceof.NewString(),
		}
	}

	// Fall through means that additional groups are selected.
	// First, get all keys to the Groups map
	groupIDs := make(sliceof.String, len(stream.Permissions))
	index := 0

	for groupID := range stream.Permissions {
		groupIDs[index] = groupID
		index++
	}

	return mapof.Any{
		"rule":     "private",
		"groupIds": groupIDs,
	}
}
*/
