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

// StepSetCircleSharing represents an action that can edit a top-level folder in the Domain
type StepSetCircleSharing struct {
	Method  string
	Title   string
	Message string
	Button  string
	Role    string
}

func (step StepSetCircleSharing) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetCircleSharing.Get"

	if step.Method == "post" {
		return Continue()
	}

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Builder is not a StreamBuilder"))
	}

	// Calculate the value object for this step
	value := step.calculateValue(streamBuilder._stream)
	schema := step.schema()
	element, err := step.form()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to build form for StepSetCircleSharing"))
	}

	f := form.New(schema, element)

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.Div().Class("flex-row", "flex-align-center")
	b.H2().Class("margin-none", "flex-grow").InnerText(step.Title).Close()
	b.A("/@me/settings/circles").InnerHTML("Manage Circles &rarr;").Close()
	b.Close()
	b.H3().InnerText(step.Message).Close()

	formHTML, err := f.Editor(value, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error rendering form for StepSetCircleSharing"))
	}

	b.WriteString(formHTML)
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

	if step.Method == "get" {
		return Continue()
	}

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
	stream.Circles[step.Role] = id.NewSlice()

	for _, permission := range request.Form["circles"] {

		// Verify we have a valid CircleID, then add it to the list of allowed Circles
		circleID, err := primitive.ObjectIDFromHex(permission)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid CircleID in form input"))
		}

		// If "Anonymous" permissions are allowed, then that's all we need.
		if circleID == model.MagicGroupIDAnonymous {
			stream.Circles[step.Role] = id.NewSlice()
			break
		}

		// Otherwise, add the CircleID to the list of allowed Circles
		stream.Circles[step.Role] = append(stream.Circles[step.Role], circleID)
	}

	// If no privileges have been set, then we allow Anonymous by default
	if len(stream.Circles[step.Role]) == 0 {
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
				"circles": schema.Array{Items: schema.String{}},
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
				Path:        "circles",
				Label:       "Share with Everyone",
				Description: "Publicly visible to everyone on the Internet, signed in or not.",
				Options: mapof.Any{
					"icon":   "globe",
					"class":  "checkbutton-public",
					"value":  model.MagicGroupIDAnonymous,
					"script": "on change if my.checked tell .checkbutton-circle set your.checked to false end else set my.checked to true",
				},
			},
			{
				Type:  "check-button-group",
				Path:  "circles",
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

func (step StepSetCircleSharing) calculateValue(stream *model.Stream) mapof.Object[id.Slice] {

	if circles := stream.Circles[step.Role]; circles.NotEmpty() {
		return mapof.Object[id.Slice]{
			"circles": circles,
		}
	}

	return mapof.Object[id.Slice]{
		"circles": id.Slice{model.MagicGroupIDAnonymous},
	}
}
