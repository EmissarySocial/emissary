package handler

import (
	"net/http"

	activitypub "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WithFunc0 is a function signature for a continuation function that requires only the domain Factory
type WithFunc0 func(ctx *steranko.Context, factory *service.Factory, session data.Session) error

// WithFunc1 is a function signature for a continuation function that requires the domain Factory and a single value
type WithFunc1[T any] func(ctx *steranko.Context, factory *service.Factory, session data.Session, value *T) error

// WithFunc2 is a function signature for a continuation function that requires the domain Factory and two values
type WithFunc2[T any, U any] func(ctx *steranko.Context, factory *service.Factory, session data.Session, value *T, value2 *U) error

// WithFunc3 is a function signature for a continuation function that requires the domain Factory and three values
type WithFunc3[T any, U any, V any] func(ctx *steranko.Context, factory *service.Factory, session data.Session, value *T, value2 *U, value3 *V) error

// WithAuthenticatedUser handles boilerplate code for requests that require a signed-in user
func WithAuthenticatedUser(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Guarantee that the user is signed in
		authorization := getAuthorization(ctx)

		if !authorization.IsAuthenticated() {
			return derp.UnauthorizedError(location, "You must be signed in to perform this action")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Unable to load User")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &user)
	})
}
func WithConnection(provider string, serverFactory *server.Factory, fn WithFunc1[model.Connection]) echo.HandlerFunc {

	const location = "handler.WithConnection"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the Connection from the database
		connectionService := factory.Connection()
		connection := model.NewConnection()

		if provider == "" {
			provider = ctx.Param("provider")
		}

		if err := connectionService.LoadByProvider(session, provider, &connection); err != nil {
			return derp.Wrap(err, location, "Unable to load Connection")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &connection)
	})
}

// WithDomain handles boilerplate code for requests that load a domain object
func WithDomain(serverFactory *server.Factory, fn WithFunc1[model.Domain]) echo.HandlerFunc {

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
		domain := factory.Domain().Get()
		return fn(ctx, factory, session, domain)
	})
}

// WithFactory handles boilerplate code for requests that require only the domain Factory
func WithFactory(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithFactory"

	return func(ctx echo.Context) error {

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		/////////////////////////////////////////////////////////
		// GET requests use a simple "read only" database session

		// Call the continuation function
		if ctx.Request().Method == http.MethodGet {

			// Create a Read only database session
			session, err := factory.Server().Session(ctx.Request().Context())

			if err != nil {
				return derp.Wrap(err, location, "Unable to open database session")
			}

			defer session.Close()

			// Create a context that wraps this echo.Context and data.Session
			sterankoContext := factory.Steranko(session).Context(ctx)

			// Execute the *actual* handler (success alleged)
			return fn(sterankoContext, factory, session)
		}

		/////////////////////////////////////////////////////
		// POST requests are wrapped in a MongoDB transaction

		// WCreate a database transaction and wrap the callback function in it.
		_, err = factory.Server().WithTransaction(ctx.Request().Context(), func(session data.Session) (any, error) {
			sterankoContext := factory.Steranko(session).Context(ctx)
			return nil, fn(sterankoContext, factory, session)
		})

		if err != nil {
			return derp.Wrap(err, location, "Unable to open database session")
		}

		// Success alleged.
		return nil
	}
}

func WithFollowing(serverFactory *server.Factory, fn WithFunc1[model.Following]) echo.HandlerFunc {

	const location = "handler.WithFollowing"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Parse the UserID from the query string
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid UserID", userID)
		}

		// Parse the Following from the query string
		followingID, err := primitive.ObjectIDFromHex(ctx.Param("followingId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid FollowingID", followingID)
		}

		// Load the following record from the database
		followingService := factory.Following()
		following := model.NewFollowing()

		if err := followingService.LoadByID(session, userID, followingID, &following); err != nil {
			return derp.Wrap(err, location, "Unable to load following record", userID, followingID)
		}

		return fn(ctx, factory, session, &following)
	})
}

func WithIdentity(serverFactory *server.Factory, fn WithFunc1[model.Identity]) echo.HandlerFunc {

	const location = "handler.WithIdentity"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

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

				if err := userService.LoadByID(session, authorization.UserID, &user); err != nil {
					return derp.Wrap(err, location, "Unable to load signed-in user")
				}

				// Load/Create an Identity for the signed-in User
				identity, err := identityService.LoadOrCreate(session, user.DisplayName, model.IdentifierTypeEmail, user.EmailAddress)

				if err != nil {
					return derp.Wrap(err, location, "Unable to load/creating Identity")
				}

				// TODO: update the signed-in authorization so we don't
				// have to hit the database all the time

				return fn(ctx, factory, session, &identity)
			}

			return ctx.Redirect(http.StatusSeeOther, "/@guest/signin")
		}

		if err := identityService.LoadByID(session, authorization.IdentityID, &identity); err != nil {
			return derp.Wrap(err, location, "Unable to load Identity")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &identity)
	})
}

func WithMerchantAccount(serverFactory *server.Factory, fn WithFunc1[model.MerchantAccount]) echo.HandlerFunc {

	const location = "handler.WithMerchantAccount"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the MerchantAccount from the database
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()
		merchantAccountToken := ctx.QueryParam("merchantAccountId")

		if err := merchantAccountService.LoadByToken(session, merchantAccountToken, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Unable to load MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &merchantAccount)
	})
}

func WithMerchantAccountJWT(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProductJWT"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

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

func WithOwner(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithAdmin"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Guarantee that the user is signed in
		authorization := getAuthorization(ctx)

		if !authorization.DomainOwner {
			return derp.UnauthorizedError(location, "You must be an admin to perform this action")
		}

		// Call the continuation function
		return fn(ctx, factory, session)
	})
}

func WithPrivilege(serverFactory *server.Factory, fn WithFunc2[model.Identity, model.Privilege]) echo.HandlerFunc {

	const location = "handler.WithPrivilege"

	return WithIdentity(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, identity *model.Identity) error {

		// Load the Privilege from the database
		privilegeService := factory.Privilege()
		privilege := model.NewPrivilege()

		privilegeID, err := primitive.ObjectIDFromHex(ctx.Param("privilegeId"))

		if err != nil {
			return derp.BadRequestError(location, "Invalid PrivilegeID", "PrivilegeID must be a valid ObjectID")
		}

		if err := privilegeService.LoadByIdentity(session, identity.IdentityID, privilegeID, &privilege); err != nil {
			return derp.Wrap(err, location, "Unable to load Privilege")
		}

		// Call the continuation function
		return fn(ctx, factory, session, identity, &privilege)
	})
}

// WithProduct handles boilerplate code for requests that use a Product object
func WithProduct(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProduct"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the Product from the URL parameters
		productService := factory.Product()
		product := model.NewProduct()

		if err := productService.LoadByToken(session, ctx.QueryParam("productId"), &product); err != nil {
			return derp.Wrap(err, location, "Unable to load Product")
		}

		// Load the MerchantAccount used for the Product
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()

		if err := merchantAccountService.LoadByID(session, product.MerchantAccountID, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Unable to load MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &merchantAccount, &product)
	})
}

// WithRegistration handles boilerplate code for requests that use a Registration object
func WithRegistration(serverFactory *server.Factory, fn WithFunc2[model.Domain, model.Registration]) echo.HandlerFunc {

	const location = "handler.WithRegistration"

	return WithDomain(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, domain *model.Domain) error {

		// Require that a registration form has been defined
		if !domain.HasRegistrationForm() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Try to load a (populated) Registration object from the factory
		registrationService := factory.Registration()
		registration, err := registrationService.Load(domain.RegistrationID)

		if err != nil {
			return derp.Wrap(err, location, "Unable to load Registration")
		}

		if registration.IsZero() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Call the continuation function
		return fn(ctx, factory, session, domain, &registration)
	})
}

// WithSearchQuery handles boilerplate code for requests that load a search query
func WithSearchQuery(serverFactory *server.Factory, fn WithFunc3[model.Template, model.Stream, model.SearchQuery]) echo.HandlerFunc {

	const location = "handler.WithSearchQuery"

	return WithTemplate(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

		// Load the Stream from the database
		searchQueryService := factory.SearchQuery()
		token := ctx.Param("searchId")

		switch token {

		// If there is no token, make a new token using the URL parameters provided
		case "":
			searchQuery, err := searchQueryService.LoadOrCreate(session, ctx.QueryParams())

			if err != nil {
				return derp.Wrap(err, location, "Unable to create search query token")
			}

			// Call the continuation function
			return fn(ctx, factory, session, template, stream, &searchQuery)

		// If we have a valid token, then use it to  look up the search query
		default:
			searchQuery := model.NewSearchQuery()
			if err := searchQueryService.LoadByToken(session, token, &searchQuery); err != nil {
				return derp.Wrap(err, location, "Unable to load search query from database")
			}

			// Call the continuation function
			return fn(ctx, factory, session, template, stream, &searchQuery)
		}
	})
}

// WithStream handles boilerplate code for requests that load a Stream
func WithStream(serverFactory *server.Factory, fn WithFunc1[model.Stream]) echo.HandlerFunc {

	const location = "handler.WithStream"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the Stream from the database
		streamService := factory.Stream()
		stream := model.NewStream()
		token := getStreamToken(ctx)

		// Try to load the Stream using a Token
		err := streamService.LoadByToken(session, token, &stream)

		if err == nil {
			return fn(ctx, factory, session, &stream)
		}

		// Anything but a "Not Found" error is a problem
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Unable to load stream from database")
		}

		// If the "home" page is requested but not found, then we're in "startup" mode
		if token == "home" {
			return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
		}

		// Maybe we're looking for a User, but forgot the "@" prefix?
		user := model.NewUser()
		if err := factory.User().LoadByUsername(session, token, &user); err == nil {
			return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
		}

		// I give up, man..
		return ctx.NoContent(http.StatusNotFound)
	})
}

// WithTemplate handles boilerplate code for requests that load a Stream and its corresponding Template
func WithTemplate(serverFactory *server.Factory, fn WithFunc2[model.Template, model.Stream]) echo.HandlerFunc {

	const location = "handler.WithTemplate"

	return WithStream(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {

		// Load the Stream from the database
		template, err := factory.Template().Load(stream.TemplateID)

		if err != nil {
			return derp.Wrap(err, location, "Template is not defined", stream.TemplateID)
		}

		// Call the continuation function
		return fn(ctx, factory, session, &template, stream)
	})
}

// WithUser handles boilerplate code for requests that load a user by username or ID
func WithUser(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		if err := userService.LoadByToken(session, userID, &user); err != nil {
			return derp.Wrap(err, location, "Unable to load User")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &user)
	})
}

// WithUserForwarding handles boilerplate code for requests that load a user by username or ID
// and, when called with a UserID/objectId, forwards to the user's correct username
func WithUserForwarding(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithUserForwarding"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		if err := userService.LoadByToken(session, userID, &user); err != nil {
			return derp.Wrap(err, location, "Unable to load user from database")
		}

		// If this is a JSON-LD request, then skip the forwarding and just return the User
		if isJSONLDRequest(ctx) {
			return activitypub.RenderProfileJSONLD(ctx, factory, session, &user)
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
		return fn(ctx, factory, session, &user)
	})
}
