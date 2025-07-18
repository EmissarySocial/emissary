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
func WithConnection(provider string, serverFactory *server.Factory, fn WithFunc1[model.Connection]) echo.HandlerFunc {

	const location = "handler.WithConnection"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the Connection from the database
		connectionService := factory.Connection()
		connection := model.NewConnection()

		if provider == "" {
			provider = ctx.Param("provider")
		}

		if err := connectionService.LoadByProvider(provider, &connection); err != nil {
			return derp.Wrap(err, location, "Error loading Connection")
		}

		// Call the continuation function
		return fn(ctx, factory, &connection)
	})
}

// WithDomain handles boilerplate code for requests that load a domain object
func WithDomain(serverFactory *server.Factory, fn WithFunc1[model.Domain]) echo.HandlerFunc {

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {
		domain := factory.Domain().Get()
		return fn(ctx, factory, domain)
	})
}

// WithFactory handles boilerplate code for requests that require only the domain Factory
func WithFactory(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return func(ctx echo.Context) error {

		// Cast the context to a Steranko Context
		sterankoContext, ok := ctx.(*steranko.Context)

		if !ok {
			return derp.InternalError(location, "Context must be a Steranko Context")
		}

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Call the continuation function
		return fn(sterankoContext, factory)
	}
}

func WithIdentity(serverFactory *server.Factory, fn WithFunc1[model.Identity]) echo.HandlerFunc {

	const location = "handler.WithIdentity"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		identityService := factory.Identity()
		identity := model.NewIdentity()

		authorization := getAuthorization(ctx)

		if authorization.IdentityID.IsZero() {

			// If we're authenticated but don't have an Identity,
			// then we'll MAKE one using the authenticated user
			if authorization.IsAuthenticated() {

				// Load the signed-in user
				userService := factory.User()
				user := model.NewUser()

				if err := userService.LoadByID(authorization.UserID, &user); err != nil {
					return derp.Wrap(err, location, "Error loading signed-in user")
				}

				// Load/Create an Identity for the signed-in User
				identity, err := identityService.LoadOrCreate(user.DisplayName, model.IdentifierTypeEmail, user.EmailAddress)

				if err != nil {
					return derp.Wrap(err, location, "Error loading/creating Identity")
				}

				// TODO: update the signed-in authorization so we don't
				// have to hit the database all the time

				return fn(ctx, factory, &identity)
			}

			return ctx.Redirect(http.StatusSeeOther, "/@guest/signin")
		}

		if err := identityService.LoadByID(authorization.IdentityID, &identity); err != nil {
			return derp.Wrap(err, location, "Error loading Identity")
		}

		// Call the continuation function
		return fn(ctx, factory, &identity)
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

func WithMerchantAccountJWT(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProductJWT"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Parse the JWT token from the Request
		jwtService := factory.JWT()
		claims := jwt.MapClaims{}

		if err := jwtService.ParseToken(ctx.QueryParam("jwt"), &claims); err != nil {
			return derp.Wrap(err, location, "Error parsing JWT token")
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
		ctx.QueryParams().Set("productId", productID)
		ctx.QueryParams().Set("transactionId", transactionID)

		// Continue processing using WithMerchantAccount
		return WithProduct(serverFactory, fn)(ctx)
	})
}

func WithPrivilege(serverFactory *server.Factory, fn WithFunc2[model.Identity, model.Privilege]) echo.HandlerFunc {

	const location = "handler.WithPrivilege"

	return WithIdentity(serverFactory, func(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity) error {

		// Load the Privilege from the database
		privilegeService := factory.Privilege()
		privilege := model.NewPrivilege()

		privilegeID, err := primitive.ObjectIDFromHex(ctx.Param("privilegeId"))

		if err != nil {
			return derp.BadRequestError(location, "Invalid PrivilegeID", "PrivilegeID must be a valid ObjectID")
		}

		if err := privilegeService.LoadByIdentity(identity.IdentityID, privilegeID, &privilege); err != nil {
			return derp.Wrap(err, location, "Error loading Privilege")
		}

		// Call the continuation function
		return fn(ctx, factory, identity, &privilege)
	})
}

// WithProduct handles boilerplate code for requests that use a Product object
func WithProduct(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Load the Product from the URL parameters
		productService := factory.Product()
		product := model.NewProduct()

		if err := productService.LoadByToken(ctx.QueryParam("productId"), &product); err != nil {
			return derp.Wrap(err, location, "Error loading Product")
		}

		// Load the MerchantAccount used for the Product
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()

		if err := merchantAccountService.LoadByID(product.MerchantAccountID, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Error loading MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, &merchantAccount, &product)
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

		// Try to load the Stream using a Token
		err := streamService.LoadByToken(token, &stream)

		if err == nil {
			return fn(ctx, factory, &stream)
		}

		// Anything but a "Not Found" error is a problem
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Error loading stream from database")
		}

		// If the "home" page is requested but not found, then we're in "startup" mode
		if token == "home" {
			return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
		}

		// Maybe we're looking for a User, but forgot the "@" prefix?
		user := model.NewUser()
		if err := factory.User().LoadByUsername(token, &user); err == nil {
			return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
		}

		// I give up, man..
		return ctx.NoContent(http.StatusNotFound)
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
