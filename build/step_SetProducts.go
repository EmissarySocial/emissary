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
	merchantAccountService := streamBuilder._factory.MerchantAccount()

	attributedToID := streamBuilder._stream.AttributedTo.UserID
	merchantAccounts, products, err := merchantAccountService.ProductsByUser(attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving products"))
	}

	// If there are no products, then display the "empty" message
	if len(products) == 0 {
		return step.GetEmpty(merchantAccounts, iconFunc, buffer)
	}

	roles := streamBuilder._template.PurchasableRoles()

	tabLabel := html.New()

	for _, merchantAccount := range merchantAccounts {

		tabLabel.A(merchantAccount.ProductURL()).
			Attr("target", "_blank").
			Class("nowrap", "margin-right").
			InnerHTML("Edit Products in " + merchantAccount.Name + " " + iconFunc("new-window")).
			Close()
	}

	formDefinition := form.Element{
		Type: "layout-tabs",
		Children: slice.Map(roles, func(role model.Role) form.Element {
			return form.Element{
				Type:  "layout-vertical",
				Label: role.Label,
				Children: []form.Element{
					{
						Type:        "multiselect",
						Label:       role.Description,
						Path:        role.RoleID,
						Description: tabLabel.String(),
						Options: mapof.Any{
							"rows": 10,
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

	b.Div().Class("margin-bottom-lg").InnerHTML(`
		Now that your merchant account is connected, you can select the products that grant access to this item.
		Visitors purchase access to your content, with either one-time, or recurring payments.
		<a href="https://emissary.dev/products" target="_blank" class="nowrap">` + iconFunc("help") + ` Help with Products</a>`,
	).Close()

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
	b.A(streamBuilder._stream.URL).Class("button").InnerText("Cancel").Close()
	b.Span().ID("htmx-response-message").Class("margin-left", "text-green").Close()
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

func (step StepSetProducts) GetEmpty(merchantAccounts sliceof.Object[model.MerchantAccount], iconFunc func(string) string, buffer io.Writer) PipelineBehavior {

	// Write the rest of the HTML that contains the form
	b := html.New()

	// Heading
	b.Div().Class("margin-bottom")

	b.Span().InnerHTML(`Now that your merchant account is connected, you can connect products and subscription plans to this item.`).Close()
	b.Span().InnerText("Click here for ")
	b.A(merchantAccounts.First().HelpURL()).
		Class("nowrap").
		InnerHTML(iconFunc("help") + ` Help with Products`).
		Close()

	b.Close()
	b.Close()

	b.Div().Class("margin-bottom-lg")
	for _, merchantAccount := range merchantAccounts {

		b.A(merchantAccount.ProductURL()).
			Attr("target", "_blank").
			Class("button", "primary").
			InnerHTML(`+ Add Products to ` + merchantAccount.Name).
			Close()
	}
	b.Close()

	b.Div().Class("margin-bottom-lg", "text-gray")
	b.Span().InnerText("When you're done, return here and ").Close()
	b.Span().Class("link", "text-nocolor").Script("on click reload() the window's location").InnerText("refresh this page").Close()
	b.Span().InnerText(" to connect products to this item.").Close()
	b.Close()

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
