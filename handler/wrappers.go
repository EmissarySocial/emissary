package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	activitypub "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WithFunc0 is a function signature for a continuation function that requires only the domain Factory
type WithFunc0 func(ctx *steranko.Context, factory *domain.Factory) error

// WithFunc1 is a function signature for a continuation function that requires the domain Factory and a single value
type WithFunc1[T any] func(ctx *steranko.Context, factory *domain.Factory, value *T) error

// WithFunc2 is a function signature for a continuation function that requires the domain Factory and two values
type WithFunc2[T any, U any] func(ctx *steranko.Context, factory *domain.Factory, value *T, value2 *U) error

// WithFunc3 is a function signature for a continuation function that requires the domain Factory and three values
type WithFunc3[T any, U any, V any] func(ctx *steranko.Context, factory *domain.Factory, value *T, value2 *U, value3 *V) error

// WithFactory handles boilerplate code for requests that require only the domain Factory
func WithFactory(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return func(ctx echo.Context) error {

		// Cast the context to a Steranko Context
		sterankoContext, ok := ctx.(*steranko.Context)

		if !ok {
			return derp.ReportAndReturn(derp.InternalError(location, "Context must be a Steranko Context"))
		}

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.ReportAndReturn(derp.Wrap(err, location, "Unrecognized Domain"))
		}

		// Call the continuation function
		return derp.ReportAndReturn(fn(sterankoContext, factory))
	}
}

// WithDomain handles boilerplate code for requests that load a domain object
func WithDomain(serverFactory *server.Factory, fn WithFunc1[model.Domain]) echo.HandlerFunc {

	const location = "handler.WithRegistration"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Try to retrieve the Domain record
		domainService := factory.Domain()
		domain, err := domainService.LoadDomain()

		if err != nil {
			return derp.Wrap(err, location, "Error loading Domain")
		}

		return fn(ctx, factory, &domain)
	})
}

func WithMerchantAccount(serverFactory *server.Factory, fn WithFunc1[model.MerchantAccount]) echo.HandlerFunc {

	const location = "handler.WithMerchantAccount"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the MerchantAccount from the database
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()
		merchantAccountToken := ctx.QueryParam("merchantAccountId")

		if err := merchantAccountService.LoadByToken(merchantAccountToken, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Error loading MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, &merchantAccount)
	})

}

// WithRegistration handles boilerplate code for requests that use a Registration object
func WithRegistration(serverFactory *server.Factory, fn WithFunc2[model.Domain, model.Registration]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithDomain(serverFactory, func(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain) error {

		// Require that a registration form has been defined
		if !domain.HasRegistrationForm() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Try to load a (populated) Registration object from the factory
		registrationService := factory.Registration()
		registration, err := registrationService.Load(domain.RegistrationID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading Registration")
		}

		if registration.IsZero() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Call the continuation function
		return fn(ctx, factory, domain, &registration)
	})
}

// WithSearchQuery handles boilerplate code for requests that load a search query
func WithSearchQuery(serverFactory *server.Factory, fn WithFunc3[model.Template, model.Stream, model.SearchQuery]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithTemplate(serverFactory, func(ctx *steranko.Context, factory *domain.Factory, template *model.Template, stream *model.Stream) error {

		// Load the Stream from the database
		searchQueryService := factory.SearchQuery()
		token := ctx.Param("searchId")

		switch token {

		// If there is no token, make a new token using the URL parameters provided
		case "":
			searchQuery, err := searchQueryService.LoadOrCreate(ctx.QueryParams())

			if err != nil {
				return derp.Wrap(err, location, "Error creating search query token")
			}

			// Call the continuation function
			return fn(ctx, factory, template, stream, &searchQuery)

		// If we have a valid token, then use it to  look up the search query
		default:
			searchQuery := model.NewSearchQuery()
			if err := searchQueryService.LoadByToken(token, &searchQuery); err != nil {
				return derp.Wrap(err, location, "Error loading search query from database")
			}

			// Call the continuation function
			return fn(ctx, factory, template, stream, &searchQuery)
		}
	})
}

// WithStream handles boilerplate code for requests that load a Stream
func WithStream(serverFactory *server.Factory, fn WithFunc1[model.Stream]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the Stream from the database
		streamService := factory.Stream()
		stream := model.NewStream()
		token := getStreamToken(ctx)

		if err := streamService.LoadByToken(token, &stream); err != nil {

			// Special case: If the HOME page is missing, then this is a new database.  Forward to the admin section
			if derp.IsNotFound(err) && (token == "home") {
				return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
			}

			return derp.Wrap(err, location, "Error loading stream from database")
		}

		// Call the continuation function
		return fn(ctx, factory, &stream)
	})
}

func WithProduct(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProduct"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Get the UserID from the the URL
		userID, err := primitive.ObjectIDFromHex(ctx.QueryParam("userId"))

		if err != nil {
			return derp.Wrap(err, location, "UserID must be a valid ObjectID", ctx.QueryParam("userId"))
		}

		// Get the ProductID from the the URL
		productID, err := primitive.ObjectIDFromHex(ctx.QueryParam("productId"))

		if err != nil {
			return derp.Wrap(err, location, "ProductID must be a valid ObjectID", ctx.QueryParam("productId"))
		}

		// Load the Product from the database
		productService := factory.Product()
		product := model.NewProduct()

		if err := productService.LoadByUserAndID(userID, productID, &product); err != nil {
			return derp.Wrap(err, location, "Error loading Product")
		}

		// Load the MerchantAccount from the database
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()

		if err := merchantAccountService.LoadByID(product.MerchantAccountID, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Error loading MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, &merchantAccount, &product)
	})
}

func WithProductJWT(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProductJWT"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Parse the JWT token from the Request
		jwtService := factory.JWT()
		claims := jwt.MapClaims{}

		if err := jwtService.ParseToken(ctx.QueryParam("jwt"), &claims); err != nil {
			return derp.Wrap(err, location, "Error parsing JWT token")
		}

		// Retrieve the MerchantAccountID
		userID, isString := claims["userId"].(string)
		if !isString {
			return derp.BadRequestError(location, "UserID in JWT token must be a string")
		}

		// Retrive the ProductID
		productID, isString := claims["productId"].(string)
		if !isString {
			return derp.BadRequestError(location, "ProductID in JWT token must be a string")
		}

		// Retrieve TransactionID (client_reference_id)
		transactionID, isString := claims["transactionId"].(string)
		if !isString {
			return derp.BadRequestError(location, "AuthorizationCode in JWT token must be a string")
		}

		// Apply the values to the context
		ctx.QueryParams().Set("userId", userID)
		ctx.QueryParams().Set("productId", productID)
		ctx.QueryParams().Set("transactionId", transactionID)

		// Continue processing using WithProduct
		return WithProduct(serverFactory, fn)(ctx)
	})
}

// WithTemplate handles boilerplate code for requests that load a Stream and its corresponding Template
func WithTemplate(serverFactory *server.Factory, fn WithFunc2[model.Template, model.Stream]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithStream(serverFactory, func(ctx *steranko.Context, factory *domain.Factory, stream *model.Stream) error {

		// Load the Stream from the database
		template, err := factory.Template().Load(stream.TemplateID)

		if err != nil {
			return derp.Wrap(err, location, "Template is not defined", stream.TemplateID)
		}

		// Call the continuation function
		return fn(ctx, factory, &template, stream)
	})
}

// WithUser handles boilerplate code for requests that load a user by username or ID
func WithUser(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User")
		}

		// Call the continuation function
		return fn(ctx, factory, &user)
	})
}

// WithUserForwarding handles boilerplate code for requests that load a user by username or ID
// and, when called with a UserID/objectId, forwards to the user's correct username
func WithUserForwarding(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithUserForwarding"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user from database")
		}

		// If this is a JSON-LD request, then skip the forwarding and just return the User
		if isJSONLDRequest(ctx) {
			return activitypub.RenderProfileJSONLD(ctx, factory, &user)
		}

		// If this is actually an objectID/userID
		if _, err := primitive.ObjectIDFromHex(userID); err == nil {

			// And guarantee that the user doesn't have a wonky username that LOOKS like a hex string
			// (for some strange reason). Then we're going to forward to the `correctURL` that uses
			// their actual username
			if user.Username != userID {

				// Build the user's correct URL
				correctURL := "/@" + user.Username

				if action := ctx.Param("action"); action != "" {
					correctURL += "/" + action
				}

				// If this is an HTMX request, then we can just update the header and continue without a full redirect
				if ctx.Request().Header.Get("Hx-Request") == "true" {
					ctx.Response().Header().Set("HX-Replace-Url", correctURL)

				} else {
					// Otherwise, we can skip the remaining code and just redirect to the correctURL
					return ctx.Redirect(http.StatusSeeOther, correctURL)
				}
			}
		}

		// Execute the continuation function
		return fn(ctx, factory, &user)
	})
}

// WithAuthenticatedUser handles boilerplate code for requests that require a signed-in user
func WithAuthenticatedUser(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Guarantee that the user is signed in
		authorization := getAuthorization(ctx)

		if !authorization.IsAuthenticated() {
			return derp.UnauthorizedError(location, "You must be signed in to perform this action")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User")
		}

		// Call the continuation function
		return fn(ctx, factory, &user)
	})
}
