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
	"github.com/benpate/rosetta/sliceof"
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

	iconFunc := streamBuilder._factory.Icons().Get

	// Load the User's Subscriptions
	attributedToID := streamBuilder._stream.AttributedTo.UserID
	subscriptions, err := streamBuilder._factory.Subscription().QueryAsLookupCodes(attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving subscriptions"))
	}

	// If there are no subscriptions, then display the "empty" message
	if len(subscriptions) == 0 {
		return step.GetEmpty(iconFunc, buffer)
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
						Type:  "multiselect",
						Label: role.Description,
						Path:  role.RoleID,
						Options: mapof.Any{
							"enum": subscriptions,
						},
					},
				},
			}
		}),
	}

	// Try to write form HTML
	formHTML, err := form.Editor(
		step.schema(streamBuilder._template.SubscribableRoles()),
		formDefinition,
		streamBuilder._stream.Subscriptions,
		builder.lookupProvider(),
	)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSubscriptions.Get", "Error building form"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().ID("modal-title").InnerText(step.Title).Close()
	b.Div().Class("alert-blue margin-bottom-lg")
	{
		b.Div().InnerHTML(`
			Subscriptions let visitors purchase access to your content, with either one-time, or recurring payments.
			<a href="https://emissary.dev/subscriptions" target="_blank">Learn more about subscriptions ` + iconFunc("new-window") + `</a>
			<br>
			<br>
			<a href="/@me/inbox/subscriptions">Edit My Subscriptions &rarr;</a>`).Close()
	}
	b.Close()

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

	const location = "build.StepSetSubscriptions.Post"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)
	if !isStreamBuilder {
		return Halt().WithError(derp.NewBadRequestError(location, "Invalid builder type"))
	}

	// Try to parse the form input
	request := streamBuilder.request()

	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetSubscriptions", "Error parsing form input"))
	}

	// Clear out existing subscription settings
	streamBuilder._stream.Subscriptions = mapof.NewObject[sliceof.String]()

	// Apply new subscription settings
	for roleID, subscriptionIDs := range request.Form {

		// Ensure that the roleID exists in the stream.Subscriptions
		if _, ok := streamBuilder._stream.Subscriptions[roleID]; !ok {
			streamBuilder._stream.Subscriptions[roleID] = sliceof.NewString()
		}

		// Append the roleId to the stream.Subscriptions
		streamBuilder._stream.Subscriptions[roleID] = append(streamBuilder._stream.Subscriptions[roleID], subscriptionIDs...)
	}

	// Success!
	return nil
}

func (step StepSetSubscriptions) GetEmpty(iconFunc func(string) string, buffer io.Writer) PipelineBehavior {

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().ID("modal-title").InnerText(step.Title).Close()
	b.Div().Class("margin-bottom-lg")
	{
		b.Div().Class("margin-bottom").InnerHTML(`
			Visitors can pay for access to this stream using <b>subscriptions</b>, which are paid directly to your own <b>merchant account</b>.
			<a href="https://emissary.dev/subscriptions" target="_blank">Learn more ` + iconFunc("new-window") + `</a>
		`).Close()
		b.Div().Class("margin-bottom").InnerHTML(`
			To get started, you'll need to set up at least one subscription plan, then return here to link it to this stream.
		`).Close()
	}
	b.Close()

	b.Button().Script("on click go to url /@me/inbox/subscriptions").Class("primary").InnerHTML("Add a New Subscription &rarr;").Close()
	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()
	b.CloseAll()

	// nolint:errcheck
	io.WriteString(buffer, b.String())
	return nil
}

// schema returns the validating schema for this form
func (step StepSetSubscriptions) schema(roles []model.Role) schema.Schema {

	properties := map[string]schema.Element{}

	for _, role := range roles {
		properties[role.RoleID] = schema.Array{Items: schema.String{Format: "objectId"}}
	}

	return schema.Schema{
		Element: schema.Object{
			Properties: properties,
		},
	}
}
