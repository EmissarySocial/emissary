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
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// StepSetPrivileges represents an action that can edit a top-level folder in the Domain
type StepSetPrivileges struct {
	Title string
}

func (step StepSetPrivileges) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepSetPrivileges.Get"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Invalid builder type"))
	}

	// Collect prerequisites
	factory := streamBuilder.factory()
	attributedToID := streamBuilder._stream.AttributedTo.UserID
	iconFunc := factory.Icons().Get

	// Load the Products for this User
	merchantAccounts, products, err := factory.Product().SyncRemoteProducts(builder.session(), attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving products"))
	}

	if merchantAccounts.IsEmpty() {
		return step.GetEmpty(merchantAccounts, iconFunc, buffer)
	}

	// Load the Circles defined by this User
	circles, err := factory.Circle().QueryByUser(builder.session(), attributedToID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error retrieving circles"))
	}

	// If there are no multi-select options, then display the "empty" message
	if circles.IsEmpty() && products.IsEmpty() {
		return step.GetEmpty(merchantAccounts, iconFunc, buffer)
	}

	// Fall through means that we display the selection form

	editLinks := html.New()

	for _, merchantAccount := range merchantAccounts {

		editLinks.A(merchantAccount.ProductURL()).
			Attr("target", "_blank").
			Class("nowrap", "margin-right").
			InnerHTML("Manage Products in " + merchantAccount.Name + " &rarr;").
			Close()
	}

	roles := streamBuilder._template.PrivilegedRoles()

	formDefinition := form.Element{
		Type: "layout-tabs",
		Children: slice.Map(roles, func(role model.Role) form.Element {
			return form.Element{
				Type:        "layout-vertical",
				Label:       role.Label,
				Description: role.Description,
				Children: []form.Element{
					{
						Type:        "multiselect",
						Label:       "Circles",
						Path:        "circles." + role.RoleID,
						Description: `<a href="/@me/settings/circles" target="_blank">Manage Circles &rarr;</a>`,
						Options: mapof.Any{
							"rows": 6,
							"enum": mapCirclesToLookupCodes(circles...),
						},
					},
					{
						Type:        "multiselect",
						Label:       "Products",
						Path:        "products." + role.RoleID,
						Description: editLinks.String(),
						Options: mapof.Any{
							"rows": 8,
							"enum": mapProductsToLookupCodes(products...),
						},
					},
				},
			}
		}),
	}

	// Write form HTML
	formHTML, err := form.Editor(
		step.schema(streamBuilder._template.PrivilegedRoles()),
		formDefinition,
		streamBuilder._stream,
		builder.lookupProvider(),
	)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetPrivileges.Get", "Unable to build form"))
	}

	// Write the rest of the HTML that contains the form
	b := html.New()

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

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing form HTML to buffer"))
	}

	return nil
}

func (step StepSetPrivileges) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetPrivileges.Post"

	// This step can only be used with a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)
	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequestError(location, "Invalid builder type"))
	}

	// Try to parse the form input
	request := streamBuilder.request()

	if err := request.ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetPrivileges", "Error parsing form input"))
	}

	// Clear out existing product settings
	stream := streamBuilder._stream
	stream.Circles = mapof.NewObject[id.Slice]()
	stream.Products = mapof.NewObject[id.Slice]()

	// Apply new product settings
	for key, values := range request.Form {

		valueIDs, err := id.ConvertSlice(values)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error converting form values to IDs"))
		}

		property, role, _ := strings.Cut(key, ".")

		if role == "" {
			return Halt().WithError(derp.BadRequestError(location, "Role must not be empty", key, values))
		}

		switch property {

		case "circles":
			stream.Circles[role] = valueIDs

		case "products":
			stream.Products[role] = valueIDs

		default:
			return Halt().WithError(derp.BadRequestError(location, "Property must be 'circles' or 'products'", key, values))
		}
	}

	// Success!
	return nil
}

func (step StepSetPrivileges) GetEmpty(merchantAccounts sliceof.Object[model.MerchantAccount], iconFunc func(string) string, buffer io.Writer) PipelineBehavior {

	// Write the rest of the HTML that contains the form
	b := html.New()

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

	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSetPrivileges.GetEmpty", "Error writing empty form HTML to buffer"))
	}

	return nil
}

// schema returns the validating schema for this form
func (step StepSetPrivileges) schema(roles []model.Role) schema.Schema {

	properties := map[string]schema.Element{}

	for _, role := range roles {
		properties[role.RoleID] = schema.Array{Items: schema.String{Format: "objectId"}}
	}

	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"circles":  schema.Object{Properties: properties},
				"products": schema.Object{Properties: properties},
			},
		},
	}
}
