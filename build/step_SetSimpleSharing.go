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
		return Halt().WithError(derp.BadRequest(location, "Builder is not a StreamBuilder"))
	}

	// Calculate the value object for this step
	value := step.calculateValue(streamBuilder._stream)
	schema := step.schema()
	element, err := step.form()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to build form for StepSetSimpleSharing"))
	}

	form := form.New(schema, element)

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

	formHTML, err := form.Editor(value, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error rendering form for StepSetSimpleSharing"))
	}

	b.WriteString(formHTML)
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
		return Halt().WithError(derp.BadRequest(location, "Builder is not a StreamBuilder"))
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
