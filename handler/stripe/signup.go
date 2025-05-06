package stripe

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
	"github.com/stripe/stripe-go/v78/webhook"
)

func PostSignupWebhook(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain) error {

	const location = "handler.stripe.PostWebhook"

	//////////////////////////////////////
	// 1. PREPARE AND VALIDATE THE REQUEST

	// RULE: Require that a registration form has been defined
	if !domain.HasRegistrationForm() {
		return derp.ReportAndReturn(derp.NotFoundError(location, "Stripe Webhook not defined (no registration form)"))
	}

	// Collect Registration Metadata
	secret := domain.RegistrationData.GetString("stripe_webhook_secret")
	if secret == "" {
		return derp.ReportAndReturn(derp.InternalError(location, "Stripe Webhook Secret not defined"))
	}

	restrictedKey := domain.RegistrationData.GetString("stripe_restricted_key")
	if restrictedKey == "" {
		return derp.ReportAndReturn(derp.InternalError(location, "Stripe Restricted Key not defined"))
	}

	////////////////////////////////
	// 2. READ DATA FROM THE WEBHOOK

	// Read the request body
	payload, err := io.ReadAll(ctx.Request().Body)

	if err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error reading request body"))
	}

	// Verify the WebHook signature
	signatureHeader := ctx.Request().Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, signatureHeader, secret)

	if err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error verifying webhook signature"))
	}

	// Require that the event is a "product" event
	eventType := string(event.Type)

	if !strings.HasPrefix(eventType, "customer.product.") {
		log.Trace().Str("event", eventType).Msg("Ignoring Stripe Webhook")
		return nil
	}

	log.Trace().Str("event", eventType).Msg("Processing Stripe Webhook")

	////////////////////////////////
	// 3. DRAW THE REST OF THE OWL HERE
	// Moved to an async function so that our Webhook will respond to the server quickly.
	// Whatever else happens, it's on us from here on out.
	derp.Report(finishWebhook(factory, restrictedKey, event))

	// Success?
	return ctx.NoContent(http.StatusOK)
}

func finishWebhook(factory *domain.Factory, restrictedKey string, event stripe.Event) error {

	const location = "handler.stripe.finishWebhook"

	// Get the subscription from the event details
	subscription := stripe.Subscription{}

	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return derp.Wrap(err, "handler.getProduct", "Error unmarshalling event data")
	}

	// This is the price that was paid, but it doesn't include the metadata we need.
	// So, use the API to look up the subscriptionID first.

	price := getSubscriptionPrice(&subscription)

	if price == nil {
		return derp.BadRequestError(location, "No price found in subscription", subscription)
	}

	if err := loadStripeProduct(restrictedKey, price.Product); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error getting product details"))
	}

	// Get ready to create/update a user
	userService := factory.User()
	user := model.NewUser()

	switch subscription.Status {

	// If the subscription is ACTIVE, then add the user and their group memberships
	case stripe.SubscriptionStatusActive,
		stripe.SubscriptionStatusTrialing:

		// Try to load/create the user
		if err := loadOrCreateUser(restrictedKey, userService, subscription.Customer, &user); err != nil {
			return derp.Wrap(err, location, "Error creating customer", subscription.Customer)
		}

		// Add the user to the designated groups
		addGroups(factory, &user, price.Product, "add_groups")

		// Remove the user from the designated groups
		removeGroups(factory, &user, price.Product, "remove_groups")

		// Set the user to "public" (if indicated by the Product metadata)
		setPublic(&user, price.Product, true)

	// Otherwise, CANCEL the user's product
	default:

		// If the user doesn't exists, then we don't have to cancel their access here.
		if err := loadUser(userService, subscription.Customer, &user); err != nil {
			return nil
		}

		// Since this product is no longer active, remove the user from the designated groups
		removeGroups(factory, &user, price.Product, "add_groups")

		// Set the user to "private" (if indicated by the Product metadata)
		setPublic(&user, price.Product, false)
	}

	// Save the new User to the database.  Yay!
	if err := userService.Save(&user, "Created by Stripe Webhook"); err != nil {
		return derp.Wrap(err, location, "Error saving user record")
	}

	// Success!
	return nil
}

// addGroups adds groups to the User's list, as specified by the Product metadata
func addGroups(factory *domain.Factory, user *model.User, product *stripe.Product, token string) {

	if user == nil {
		return
	}

	if product == nil {
		return
	}

	groupService := factory.Group()
	groupIDs := strings.Split(product.Metadata[token], ",")

	for _, groupToken := range groupIDs {
		group := model.NewGroup()
		if err := groupService.LoadByToken(groupToken, &group); err == nil {
			user.AddGroup(group.GroupID)
		}
	}
}

// removeGroups removes groups from the User's list, as specified by the Product metadata
func removeGroups(factory *domain.Factory, user *model.User, product *stripe.Product, token string) {

	if user == nil {
		return
	}

	if product == nil {
		return
	}

	groupService := factory.Group()
	groupIDs := strings.Split(product.Metadata[token], ",")

	for _, groupToken := range groupIDs {
		group := model.NewGroup()
		if err := groupService.LoadByToken(groupToken, &group); err == nil {
			user.RemoveGroup(group.GroupID)
		}
	}
}

// setPublic sets the User's "IsPublic" flag to the specified value (if indicated by the Product metadata)
func setPublic(user *model.User, product *stripe.Product, value bool) {

	if user == nil {
		return
	}

	if product == nil {
		return
	}

	if setPublic := product.Metadata["set_public"]; setPublic == "true" {
		user.IsPublic = value
	}
}

func getSubscriptionPrice(subscription *stripe.Subscription) *stripe.Price {

	if items := subscription.Items; items != nil {
		for _, item := range items.Data {
			if item.Price != nil {
				return item.Price
			}
		}
	}

	return nil
}

func loadUser(userService *service.User, customer *stripe.Customer, user *model.User) error {

	if customer == nil {
		return derp.BadRequestError("handler.stripe.loadUser", "Customer must not be nil")
	}

	// Try to load the user by their email address
	if err := userService.LoadByMapID(model.UserMapIDStripe, customer.ID, user); err != nil {
		return derp.Wrap(err, "handler.stripe.loadUser", "Error loading user record")
	}

	return nil
}

func loadOrCreateUser(apiKey string, userService *service.User, customer *stripe.Customer, user *model.User) error {

	err := loadUser(userService, customer, user)

	if err == nil {
		return nil
	}

	if derp.IsNotFound(err) {

		if err := loadStripeCustomer(apiKey, customer); err != nil {
			return derp.Wrap(err, "handler.stripe.loadOrCreateUser", "Error loading customer from Stripe API")
		}

		if customer.Name != "" {
			user.DisplayName = customer.Name
		} else if customer.Description != "" {
			user.DisplayName = customer.Description
		}

		user.EmailAddress = customer.Email
		user.MapIDs[model.UserMapIDStripe] = customer.ID

		return nil
	}

	return derp.Wrap(err, "handler.stripe.loadOrCreateUser", "Error loading user record")
}

func loadStripeCustomer(apiKey string, customer *stripe.Customer) error {

	const location = "handler.stripe.loadStripeCustomer"

	if customer == nil {
		return derp.BadRequestError(location, "Customer must not be nil")
	}

	if customer.ID == "" {
		return derp.BadRequestError(location, "Customer.ID must not be empty")
	}

	// Create an API client
	stripeClient := client.API{}
	stripeClient.Init(apiKey, nil)

	// Load the Customer
	params := stripe.CustomerParams{}
	value, err := stripeClient.Customers.Get(customer.ID, &params)

	if err != nil {
		return derp.Wrap(err, location, "Error loading customer from Stripe API")
	}

	// Copy the value from the API call into the original customer
	*customer = *value

	// Success
	return nil
}

func loadStripeProduct(apiKey string, product *stripe.Product) error {

	const location = "handler.stripe.loadStripeProduct"

	if product == nil {
		return derp.BadRequestError(location, "Product must not be nil")
	}

	if product.ID == "" {
		return derp.BadRequestError(location, "Product.ID must not be empty")
	}

	// Create an API client
	stripeClient := client.API{}
	stripeClient.Init(apiKey, nil)

	// Load the Product
	params := stripe.ProductParams{}
	value, err := stripeClient.Products.Get(product.ID, &params)

	if err != nil {
		return derp.Wrap(err, location, "Error loading product from Stripe API")
	}

	// Copy the value from the API call into the original product
	*product = *value

	// Success
	return nil
}
