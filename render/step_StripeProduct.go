package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"github.com/whisperverse/whisperverse/model"
)

// StepStripeProduct represents an action-step that forwards the user to a new page.
type StepStripeProduct struct {
	Title string
}

func (step StepStripeProduct) UseGlobalWrapper() bool {
	return false
}

func (step StepStripeProduct) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepStripeProduct.Get"

	factory := renderer.factory()
	formLibrary := factory.FormLibrary()
	s := stepStripeProductTransaction{}.schema()
	s = schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"data": s.Element,
			},
		},
	}

	api, err := factory.StripeClient()

	if err != nil {
		return derp.Wrap(err, location, "Error getting Stripe API client")
	}

	f := form.Form{
		Kind:  "layout-tabs",
		Label: step.Title,
		Children: []form.Form{
			{
				Kind:  "layout-vertical",
				Label: "Product",
				Children: []form.Form{
					{Kind: "text", Label: "Product Name", Path: "data.productName", Description: "Displayed on Stripe dashboard.  Not visible to visitors"},
					{Kind: "text", Label: "Price", Path: "data.decimalAmount", Options: datatype.Map{"step": 0.01}},
					{Kind: "select", Label: "Tax Rate", Path: "data.taxId", Description: "Sign in to your Stripe account to manage tax rates.", Options: datatype.Map{"options": step.getTaxRates(api)}},
					{Kind: "select", Label: "Shipping Method", Description: "Sign in to your Stripe account to manage shipping options.", Path: "data.shippingMethod", Options: datatype.Map{"options": step.getShippingMethods(api)}},
					{Kind: "text", Label: "Buy Button Label", Path: "data.buttonLabel"},
					{Kind: "toggle", Label: "", Path: "data.active", Options: datatype.Map{"true-text": "Visible to Public", "false-text": "Hide from Public"}},
					{Kind: "hidden", Path: "data.productId"},
					{Kind: "hidden", Path: "data.priceId"},
				},
			},
			{
				Kind:  "layout-vertical",
				Label: "Inventory",
				Children: []form.Form{
					{Kind: "toggle", Label: "", Path: "data.trackInventory", Options: datatype.Map{"true-text": "Track inventory for this item", "false-text": "Do not track inventory"}},
					{Kind: "text", Label: "Available Quantity", Path: "data.quantityOnHand", Description: "Purchases disabled when quantity reaches zero.", Show: form.Rule{Path: "data.trackInventory", Value: "'true'"}},
				},
			},
			{
				Kind:  "layout-vertical",
				Label: "Success Page",
				Children: []form.Form{
					{
						Kind:        "wysiwyg",
						Path:        "data.successHTML",
						Description: "Displayed when visitors complete a purchase.",
					},
				},
			},
		},
	}

	result, err := f.HTML(formLibrary, &s, renderer.object())

	if err != nil {
		return derp.Wrap(err, location, "Error rendering form")
	}

	result = WrapForm(renderer.URL(), result)

	if _, err := buffer.Write([]byte(result)); err != nil {
		return derp.Wrap(err, location, "Error writing to buffer")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStripeProduct) Post(renderer Renderer) error {

	const location = "render.StepStripeProduct.Post"

	// Collect top-level services
	factory := renderer.factory()
	streamService := factory.Stream()
	transaction := stepStripeProductTransaction{}

	// Collect and validate transaction from the request body
	if err := renderer.context().Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Error binding request body")
	}

	if err := transaction.validate(); err != nil {
		return derp.Wrap(err, location, "Error setting values", transaction)
	}

	// Connect to the stripe API
	api, err := factory.StripeClient()

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving Stripe Client")
	}

	// Confirm that a product exists
	if transaction.ProductID == "" {
		p, err := api.Products.New(&stripe.ProductParams{
			Name:   stripe.String(transaction.ProductName),
			Active: stripe.Bool(transaction.Active),
		})

		if err != nil {
			return derp.Wrap(err, location, "Error creating Stripe product", transaction)
		}

		transaction.ProductID = p.ID

	} else {

		_, err := api.Products.Update(transaction.ProductID, &stripe.ProductParams{
			Name:   stripe.String(transaction.ProductName),
			Active: stripe.Bool(transaction.Active),
		})

		if err != nil {
			return derp.Wrap(err, location, "Error updating Stripe product", transaction)
		}
	}

	// Determine if the price has changed or not
	if step.priceChanged(api, transaction.PriceID, transaction.unitAmount()) {

		// Delete old price (if needed)
		if transaction.PriceID != "" {
			_, err := api.Prices.Update(transaction.PriceID, &stripe.PriceParams{
				Active: stripe.Bool(false),
			})

			if err != nil {
				return derp.Wrap(err, location, "Error deactivating old price")
			}
		}

		// Create a new price
		p, err := api.Prices.New(&stripe.PriceParams{
			Product:    stripe.String(transaction.ProductID),
			UnitAmount: stripe.Int64(transaction.unitAmount()),
			Currency:   stripe.String(string(stripe.CurrencyUSD)),
		})

		if err != nil {
			return derp.Wrap(err, location, "Error creating Stripe price", transaction)
		}

		transaction.PriceID = p.ID
	}

	// Update the stream with the new data and save to the database
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream

	transaction.apply(stream)

	if err := streamService.Save(stream, "Stripe settings updated"); err != nil {
		return derp.Wrap(err, location, "Error saving stream")
	}

	// Send realtime update to this stream (and its parent)
	factory.StreamUpdateChannel() <- *stream

	return nil
}

// getTaxRates queries Stripe for all pre-configured tax rates and returns them as a slice of OptionCodes
func (step StepStripeProduct) getTaxRates(api client.API) []form.OptionCode {

	result := make([]form.OptionCode, 0)

	// Map all tax rates into a slice of form.OptionCode
	it := api.TaxRates.List(nil)

	for it.Next() {
		taxRate := it.TaxRate()
		result = append(result, form.OptionCode{
			Value: taxRate.ID,
			Label: taxRate.Description,
		})
	}

	// Woot woot!
	return result
}

// getShippingMethods queries Stripe for all pre-configured shipping methods and returns them as a slice of OptionCodes
func (step StepStripeProduct) getShippingMethods(api client.API) []form.OptionCode {

	result := make([]form.OptionCode, 0)

	// Map all tax rates into a slice of form.OptionCode
	it := api.ShippingRates.List(nil)

	for it.Next() {
		shippingRate := it.ShippingRate()
		result = append(result, form.OptionCode{
			Value: shippingRate.ID,
			Label: shippingRate.DisplayName,
		})
	}

	// Woot woot!
	return result
}

// priceChanged returns TRUE if the transaction abount is different from the existing price
func (step StepStripeProduct) priceChanged(api client.API, priceID string, unitAmount int64) bool {

	if priceID != "" {
		return true
	}

	price, err := api.Prices.Get(priceID, nil)

	if err != nil {
		return true
	}

	if price.UnitAmount != unitAmount {
		return true
	}

	if !price.Active {
		return true
	}

	return false
}

/*************************************
 * TRANSACTION DEFINITION
 *************************************/

// stepStripeProductTransaction collects all of the data to be updated by the StripeProduct step
type stepStripeProductTransaction struct {
	ButtonLabel    string  `path:"buttonLabel"     form:"data.buttonLabel"`
	ProductName    string  `path:"productName"     form:"data.productName"`
	DecimalAmount  float64 `path:"decimalAmount"   form:"data.decimalAmount"`
	TrackInventory bool    `path:"trackInventory"  form:"data.trackInventory"`
	QuantityOnHand int     `path:"quantityOnHand"  form:"data.quantityOnHand"`
	Active         bool    `path:"active"          form:"data.active"`
	SuccessHTML    string  `path:"successHTML"     form:"data.successHTML"`
	ShippingMethod string  `path:"shippingMethod"  form:"data.shippingMethod"`
	ProductID      string  `path:"productId"       form:"data.productId"`
	PriceID        string  `path:"priceId"         form:"data.priceId"`
	TaxID          string  `path:"taxId"           form:"data.taxId"`
}

func (txn stepStripeProductTransaction) validate() error {
	return txn.schema().Validate(txn)
}

func (txn stepStripeProductTransaction) schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"buttonLabel":    schema.String{MaxLength: null.NewInt(50), Default: "Buy Now", Required: true},
				"productName":    schema.String{MaxLength: null.NewInt(50), Required: true},
				"decimalAmount":  schema.Number{Minimum: null.NewFloat(0), Required: true},
				"trackInventory": schema.Boolean{},
				"quantityOnHand": schema.Integer{Minimum: null.NewInt64(0)},
				"active":         schema.Boolean{},
				"successHTML":    schema.String{Format: "html"},
				"shippingMethod": schema.String{},
				"taxId":          schema.String{},
				"productId":      schema.String{},
				"priceId":        schema.String{},
			},
		},
	}
}

func (txn stepStripeProductTransaction) unitAmount() int64 {
	return int64(txn.DecimalAmount * 100)
}

func (txn stepStripeProductTransaction) apply(stream *model.Stream) {

	stream.Data = datatype.Map{
		"buttonLabel":    txn.ButtonLabel,
		"productName":    txn.ProductName,
		"active":         txn.Active,
		"decimalAmount":  txn.DecimalAmount,
		"trackInventory": txn.TrackInventory,
		"quantityOnHand": txn.QuantityOnHand,
		"successHTML":    txn.SuccessHTML,
		"shippingMethod": txn.ShippingMethod,
		"productId":      txn.ProductID,
		"priceId":        txn.PriceID,
		"taxId":          txn.TaxID,
	}

	if txn.Active {
		if txn.TrackInventory && txn.QuantityOnHand == 0 {
			stream.StateID = "sold-out"
		} else {
			stream.StateID = "ready"
		}
	} else {
		stream.StateID = "new"
	}
}
