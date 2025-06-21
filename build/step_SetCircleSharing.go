package build

import (
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetCircleSharing represents an action that can edit a top-level folder in the Domain
type StepSetCircleSharing struct {
	Title   string
	Message string
	Button  string
	Role    string
}

func (step StepSetCircleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetCircleSharing.Get"

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Builder is not a StreamBuilder"))
	}

	// Calculate the value object for this step
	value := step.calculateValue(streamBuilder._stream)
	schema := step.schema()
	form, err := step.form()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error building form for StepSetCircleSharing"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H2().InnerText(step.Title).Close()
	b.H3().InnerText(step.Message).Close()

	if err := form.Edit(&schema, builder.lookupProvider(), value, b); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error rendering form for StepSetCircleSharing"))
	}

	b.CloseAll()

	result := WrapForm(builder.URL(), b.String(), "application/x-www-form-urlencoded", "submit-label:"+step.Button)

	// Write the result to the buffer
	if _, err := io.WriteString(buffer, result); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing HTML to buffer"))
	}

	return nil
}

func (step StepSetCircleSharing) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetCircleSharing.Post"

	// Guarantee that we have a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Builder is not a StreamBuilder"))
	}

	// Try to parse the form input
	request := streamBuilder.request()

	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetCircleSharing", "Error parsing form input"))
	}

	// Reset mapped privileges for the Stream
	stream := streamBuilder._stream
	stream.Groups[step.Role] = id.NewSlice()
	stream.Privileges[step.Role] = sliceof.NewString()

	for _, permission := range request.Form["permissions"] {

		// If "Anonymous" permissions are allowed, then that's all we need.
		if permission == model.MagicRoleAnonymous {
			stream.Privileges[step.Role] = sliceof.NewString()
			break
		}

		// Verify we have a valid CircleID
		if _, err := primitive.ObjectIDFromHex(permission); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid CircleID: ", permission))
		}

		// Add the CircleID to the list of allowed Privileges
		stream.Privileges[step.Role] = append(stream.Privileges[step.Role], "CIR:"+permission)
	}

	// If no privileges have been set, then we allow Anonymous by default
	if len(stream.Privileges[step.Role]) == 0 {
		stream.Groups[step.Role] = id.Slice{model.MagicGroupIDAnonymous}
	}

	// Done!
	return nil
}

// schema returns the validating schema for this form
func (step StepSetCircleSharing) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"permissions": schema.Array{Items: schema.String{}},
			},
		},
	}
}

// form returns the form to be displayed
func (step StepSetCircleSharing) form() (form.Element, error) {

	// Build the form element
	return form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{
				Type:        "check-button",
				Path:        "permissions",
				Label:       "Share with Everyone",
				Description: "Publicly visible to everyone on the Internet, signed in or not.",
				Options: mapof.Any{
					"icon":   "globe",
					"class":  "checkbutton-public",
					"value":  model.MagicRoleAnonymous,
					"script": "on change if my.checked tell .checkbutton-circle set your.checked to false end else set my.checked to true",
				},
			},
			{
				Type:  "check-button-group",
				Path:  "permissions",
				Label: "These Circles Only",
				Options: mapof.Any{
					"class":    "checkbutton-circle",
					"provider": "circles",
					"script":   "on change if my.checked then set .checkbutton-public.checked to false",
				},
			},
		},
	}, nil
}

func (step StepSetCircleSharing) calculateValue(stream *model.Stream) mapof.Object[sliceof.String] {

	permissions := sliceof.NewString()

	for _, privilege := range stream.Privileges[step.Role] {
		if strings.HasPrefix(privilege, "CIR:") {
			circleID := strings.TrimPrefix(privilege, "CIR:")
			permissions = append(permissions, circleID)
		}
	}

	if len(permissions) > 0 {
		return mapof.Object[sliceof.String]{
			"permissions": permissions,
		}
	}

	return mapof.Object[sliceof.String]{
		"permissions": sliceof.String{model.MagicRoleAnonymous},
	}
}
