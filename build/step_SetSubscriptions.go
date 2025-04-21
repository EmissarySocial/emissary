package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
)

// StepSetSubscriptions represents an action that can edit a top-level folder in the Domain
type StepSetSubscriptions struct {
	Title string
}

func (step StepSetSubscriptions) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetSubscriptions.Get"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.NewBadRequestError(location, "Invalid builder type"))
	}

	// Get Subscriptions from the database
	attributedToID := streamBuilder._stream.AttributedTo.UserID
	subscriptions, err := streamBuilder._factory.Subscription().QueryAsLookupCodes(attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving subscriptions"))
	}

	if len(subscriptions) == 0 {
		// TODO: Prompt user to create a subscription FIRST.
	}

	roles := streamBuilder._template.SubscribableRoles()

	formDefinition := form.Element{
		Type: "layout-tabs",
		Children: slice.Map(roles, func(role model.Role) form.Element {
			return form.Element{
				Type:  "layout-vertical",
				Label: role.Label,
				Children: []form.Element{
					{
						Type:        "multiselect",
						Path:        "subscriptions." + role.RoleID,
						Label:       "Select the subscriptions that grant access as a " + role.Label,
						Description: "<a href='https://emissary.dev/subscriptions' target='_blank'>Need Help?</a>",
						Options: mapof.Any{
							"enum": subscriptions,
						},
					},
				},
			}
		}),
	}

	model := mapof.NewAny()

	// Try to write form HTML
	formHTML, err := form.Editor(step.schema(), formDefinition, model, builder.lookupProvider())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSubscriptions.Get", "Error building form"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().ID("modal-title").InnerText(step.Title).Close()

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

func (step StepSetSubscriptions) Post(builder Builder, _ io.Writer) PipelineBehavior {

	/*
		const location = "build.StepSetSubscriptions.Post"
		request := builder.request()

		// Try to parse the form input
		if err := request.ParseForm(); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepSetSubscriptions", "Error parsing form input"))
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
		streamBuilder := builder.(Stream)
		stream := streamBuilder._stream
		stream.Permissions = model.NewStreamPermissions()

		for _, groupID := range groupIDs {
			for _, role := range step.Roles {
				stream.AssignPermission(role, groupID)
			}
		}
	*/
	// Success!
	return nil
}

// schema returns the validating schema for this form
func (step StepSetSubscriptions) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"rule":     schema.String{Default: "anonymous"},
				"groupIds": schema.Array{Items: schema.String{Format: "objectId"}},
			},
		},
	}
}
