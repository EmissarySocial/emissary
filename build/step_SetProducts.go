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

// StepSetProducts represents an action that can edit a top-level folder in the Domain
type StepSetProducts struct {
	Title string
}

func (step StepSetProducts) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetProducts.Get"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Invalid builder type"))
	}

	iconFunc := streamBuilder._factory.Icons().Get

	// Load the User's Products
	attributedToID := streamBuilder._stream.AttributedTo.UserID
	products, err := streamBuilder._factory.Product().QueryAsLookupCodes(attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving products"))
	}

	// If there are no products, then display the "empty" message
	if len(products) == 0 {
		return step.GetEmpty(iconFunc, buffer)
	}

	roles := streamBuilder._template.PurchasableRoles()

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
							"enum": products,
						},
					},
				},
			}
		}),
	}

	// Try to write form HTML
	formHTML, err := form.Editor(
		step.schema(streamBuilder._template.PurchasableRoles()),
		formDefinition,
		streamBuilder._stream.Products,
		builder.lookupProvider(),
	)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetProducts.Get", "Error building form"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().ID("modal-title").InnerText(step.Title).Close()
	b.Div().Class("alert-blue margin-bottom-lg")
	{
		b.Div().InnerHTML(`
			Products let visitors purchase access to your content, with either one-time, or recurring payments.
			<a href="https://emissary.dev/products" target="_blank">Learn more about products ` + iconFunc("new-window") + `</a>
			<br>
			<br>
			<a href="/@me/inbox/products">Edit My Products &rarr;</a>`).Close()
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

func (step StepSetProducts) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetProducts.Post"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)
	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Invalid builder type"))
	}

	// Try to parse the form input
	request := streamBuilder.request()

	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetProducts", "Error parsing form input"))
	}

	// Clear out existing product settings
	streamBuilder._stream.Products = mapof.NewObject[sliceof.String]()

	// Apply new product settings
	for roleID, productIDs := range request.Form {

		// Ensure that the roleID exists in the stream.Products
		if _, ok := streamBuilder._stream.Products[roleID]; !ok {
			streamBuilder._stream.Products[roleID] = sliceof.NewString()
		}

		// Append the roleId to the stream.Products
		streamBuilder._stream.Products[roleID] = append(streamBuilder._stream.Products[roleID], productIDs...)
	}

	// Success!
	return nil
}

func (step StepSetProducts) GetEmpty(iconFunc func(string) string, buffer io.Writer) PipelineBehavior {

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.H1().ID("modal-title").InnerText(step.Title).Close()
	b.Div().Class("margin-bottom-lg")
	{
		b.Div().Class("margin-bottom").InnerHTML(`
			Visitors can pay for access to this stream using <b>products</b>, which are paid directly to your own <b>merchant account</b>.
			<a href="https://emissary.dev/products" target="_blank">Learn more ` + iconFunc("new-window") + `</a>
		`).Close()
		b.Div().Class("margin-bottom").InnerHTML(`
			To get started, you'll need to set up at least one product plan, then return here to link it to this stream.
		`).Close()
	}
	b.Close()

	b.Button().Script("on click go to url /@me/inbox/products").Class("primary").InnerHTML("Add a New Product &rarr;").Close()
	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()
	b.CloseAll()

	// nolint:errcheck
	io.WriteString(buffer, b.String())
	return nil
}

// schema returns the validating schema for this form
func (step StepSetProducts) schema(roles []model.Role) schema.Schema {

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
