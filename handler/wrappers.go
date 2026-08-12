package handler

import (
	"net/http"
	"time"

	activitypub "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/sigs"
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

// HxRedirectHeader is the header that HTMX reads to force a client-side redirect
const HxRedirectHeader = "Hx-Redirect"

// WithActor resolves the actor making the request from its credentials and passes the actor's
// ID (as a string) to the continuation function. The actor identified using 1) a signed-in
// User's cookie, 2) a valid HTTP signature, or 3) neither (an empty string for anonymous).
func WithActor(serverFactory *server.Factory, fn WithFunc1[string]) echo.HandlerFunc {

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// A signed-in User's cookie identifies them by their own ActivityPub URL
		if authorization := getAuthorization(ctx); authorization.IsAuthenticated() {
			if userID := authorization.UserID; !userID.IsZero() {
				user := model.NewUser()
				if err := factory.User().LoadByID(session, userID, &user); err == nil {
					actorID := user.ActivityPubURL()
					return fn(ctx, factory, session, &actorID)
				}
			}
		}

		// A valid HTTP signature identifies a (possibly remote) actor.
		publicKeyFinder := factory.ActivityStream().PublicKeyFinder
		if signature, err := sigs.Verify(ctx.Request(), publicKeyFinder); err == nil {
			actorID := signature.ActorID()
			return fn(ctx, factory, session, &actorID)
		}

		// Fall through means the request is Anonymous
		actorID := ""
		return fn(ctx, factory, session, &actorID)
	})
}

// WithActorAndUser resolves both the requesting Actor (authenticated by HTTP signatures) and the
//
//	requested (but un-authenticated) User from the URL path
func WithActorAndUser(serverFactory *server.Factory, fn WithFunc2[string, model.User]) echo.HandlerFunc {

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
		return WithActor(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string) error {
			return WithUser(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
				return fn(ctx, factory, session, actorID, user)
			})(ctx)
		})(ctx)
	})
}

// WithAuthorizedActorAndUser resolves the requesting Actor and the requested User, and REFUSES the
// request unless the Actor is identified and welcome: anonymous requests get a 401 UNIFORMLY --
// before the target is even loaded, so probing reveals nothing -- and a requester BLOCKED by the
// target User (or the domain) gets a 404, indistinguishable from a User that does not exist. MUTE
// and LABEL rules never gate here: a muted actor must not be able to detect the mute.
func WithAuthorizedActorAndUser(serverFactory *server.Factory, fn WithFunc2[string, model.User]) echo.HandlerFunc {

	const location = "handler.WithAuthorizedActorAndUser"

	return WithActor(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string) error {

		// RULE: Anonymous requests are refused before the target is even loaded, so probing
		// reveals nothing. A capable remote server retries the 401 with a signed request.
		if *actorID == "" {
			return derp.Unauthorized(location, "Authentication required")
		}

		return WithUser(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

			// Evaluate the target User's Rules (plus admin-tier rules) against the requester
			disposition, err := factory.Rule().DispositionForKeys(session, user.UserID, model.ActorMatchKeys(*actorID), time.Now().Unix())

			// RULE: The gate fails CLOSED -- a rules-query failure must not serve a blocked actor
			if err != nil {
				return derp.Wrap(err, location, "Checking rules for authorized fetch", *actorID)
			}

			// RULE: A blocked requester is refused before the continuation runs
			if refusal := authorizedActorRefusal(location, disposition); refusal != nil {
				return refusal
			}

			// Pass the vetted Actor and User to the continuation function
			return fn(ctx, factory, session, actorID, user)
		})(ctx)
	})
}

// authorizedActorRefusal maps the requester's disposition to the authorized-fetch outcome: a
// blocked requester gets a 404 (identical to a User that does not exist), and everyone else --
// muted included -- gets through, because a muted actor must never detect the mute (D5).
func authorizedActorRefusal(location string, disposition model.RuleDisposition) error {

	if disposition.IsBlocked() {
		return derp.NotFound(location, "User not found")
	}

	// The velvet rope parts.
	return nil
}

// WithActorAndStream resolves both the requesting Actor (authenticated by HTTP signatures) and the
// Stream they have requested
func WithActorAndStream(serverFactory *server.Factory, fn WithFunc2[string, model.Stream]) echo.HandlerFunc {

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
		return WithActor(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, actorID *string) error {
			return WithStream(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {
				return fn(ctx, factory, session, actorID, stream)
			})(ctx)
		})(ctx)
	})
}

// WithAuthenticatedAPI handles boilerplate code for requests that require a signed-in user (but does not load the User from the database)
func WithAuthenticatedAPI(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// IF the user is NOT signed in, then stop here
		if authorization := getAuthorization(ctx); authorization.NotAuthenticated() {
			return derp.Unauthorized(location, "You must be signed in to perform this action")
		}

		// Call the continuation function
		return fn(ctx, factory, session)
	})
}

// WithAuthenticatedUser handles boilerplate code for requests that require a signed-in user
func WithAuthenticatedUser(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithAuthenticatedUser"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Guarantee that the user is signed in
		authorization := getAuthorization(ctx)

		if !authorization.IsAuthenticated() {
			return derp.Unauthorized(location, "You must be signed in to perform this action")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Loading User", derp.WithUnauthorized())
		}

		// If this user has moved, then they cannot access to this server anymore.
		// Send them to their new server instead.
		if user.MovedTo != "" {
			factory.Steranko(session).SignOut(ctx)
			ctx.Response().Header().Set(HxRedirectHeader, "/signout")
			return ctx.Redirect(http.StatusTemporaryRedirect, "/signout")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &user)
	})
}

// WithConnection handles boilerplate code for requests that load a Connection object
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
			return derp.Wrap(err, location, "Loading Connection")
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
				return derp.Wrap(err, location, "Opening database session")
			}

			defer session.Close()

			// Create a context that wraps this echo.Context and data.Session
			sterankoContext := factory.Steranko(session).Context(ctx)

			// Execute the *actual* handler (success alleged)
			return fn(
				sterankoContext,
				factory,
				session,
			)
		}

		/////////////////////////////////////////////////////////
		// POST requests are wrapped in a MongoDB transaction

		// Create a database transaction and wrap the callback function in it.
		// factory.WithTransaction (not factory.Server().WithTransaction) attaches the
		// post-commit task spool, so queue tasks publish only after the commit.
		_, err = factory.WithTransaction(ctx.Request().Context(), func(session data.Session) (any, error) {
			sterankoContext := factory.Steranko(session).Context(ctx)
			return nil, fn(sterankoContext, factory, session)
		})

		if err != nil {
			return derp.Wrap(err, location, "Executing inner handler function")
		}

		// Success alleged.
		return nil
	}
}

// WithFollowing handles boilerplate code for requests that load a Following object
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
			return derp.Wrap(err, location, "Loading following record", userID, followingID)
		}

		return fn(ctx, factory, session, &following)
	})
}

// WithIdentity handles boilerplate code for requests that load (or create) the requester's Identity
func WithIdentity(serverFactory *server.Factory, fn WithFunc1[model.Identity]) echo.HandlerFunc {

	const location = "handler.WithIdentity"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		authorization := getAuthorization(ctx)

		if authorization.IdentityID.IsZero() {

			// If we're authenticated but don't have an Identity,
			// then we'll MAKE one using the authenticated user
			if authorization.IsAuthenticated() {

				// Load the signed-in user
				userService := factory.User()
				user := model.NewUser()

				if err := userService.LoadByID(session, authorization.UserID, &user); err != nil {
					return derp.Wrap(err, location, "Loading signed-in user")
				}

				// Load/Create an Identity for the signed-in User
				identityService := factory.Identity()
				identity, err := identityService.LoadOrCreate(session, user.DisplayName, model.IdentifierTypeEmail, user.EmailAddress)

				if err != nil {
					return derp.Wrap(err, location, "Loading/creating Identity")
				}

				// TODO: update the signed-in authorization so we don't
				// have to hit the database all the time

				return fn(ctx, factory, session, &identity)
			}

			return ctx.Redirect(http.StatusSeeOther, "/@guest/signin")
		}

		// Otherwise, load the Identity directly from the database
		identityService := factory.Identity()
		identity := model.NewIdentity()

		if err := identityService.LoadByID(session, authorization.IdentityID, &identity); err != nil {
			return derp.Wrap(err, location, "Loading Identity")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &identity)
	})
}

// WithMerchantAccount handles boilerplate code for requests that load a MerchantAccount object
func WithMerchantAccount(serverFactory *server.Factory, fn WithFunc1[model.MerchantAccount]) echo.HandlerFunc {

	const location = "handler.WithMerchantAccount"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Load the MerchantAccount from the database
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()
		merchantAccountToken := ctx.QueryParam("merchantAccountId")

		if err := merchantAccountService.LoadByToken(session, merchantAccountToken, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Loading MerchantAccount")
		}

		// Call the continuation function
		return fn(ctx, factory, session, &merchantAccount)
	})
}

// WithMerchantAccountJWT handles boilerplate code for requests that name their Product inside a signed JWT
func WithMerchantAccountJWT(serverFactory *server.Factory, fn WithFunc2[model.MerchantAccount, model.Product]) echo.HandlerFunc {

	const location = "handler.WithProductJWT"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Parse the JWT token from the Request
		jwtService := factory.JWT()
		claims := jwt.MapClaims{}

		if err := jwtService.ParseToken(ctx.QueryParam("jwt"), &claims); err != nil {
			return derp.Wrap(err, location, "Parsing JWT token")
		}

		// Retrive the ProductID
		productID, isString := claims["productId"].(string)
		if !isString {
			return derp.BadRequest(location, "ProductID in JWT token must be a string")
		}

		// Retrieve TransactionID (client_reference_id)
		transactionID, isString := claims["transactionId"].(string)
		if !isString {
			return derp.BadRequest(location, "AuthorizationCode in JWT token must be a string")
		}

		// Apply the values to the context
		ctx.QueryParams().Set("productId", productID)
		ctx.QueryParams().Set("transactionId", transactionID)

		// Continue processing using WithMerchantAccount
		return WithProduct(serverFactory, fn)(ctx)
	})
}

// WithOAuthUser handles boilerplate code for requests that load a user from an OAuth token
func WithOAuthUser(serverFactory *server.Factory, fn WithFunc2[model.OAuthUserToken, model.User]) echo.HandlerFunc {

	return WithAuthenticatedUser(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

		const location = "handler.WithOAuthUser"

		// RULE: Can only request routes that match the authenticated UserID
		if ctx.Param("userId") != user.UserID.Hex() {
			return derp.NotFound(location, "UserID in URL must match authenticated User")
		}

		// The access token (validated by WithAuthenticatedUser) carries the grant ID.
		grantID := getAuthorization(ctx).OAuthUserTokenID

		if grantID.IsZero() {
			return derp.Unauthorized(location, "Access token is not an OAuth grant")
		}

		// Load the grant by ID. A missing record means the grant was revoked, so the
		// (still-signed) access token is no longer honored.
		oauthUserTokenService := factory.OAuthUserToken()
		oauthUserToken := model.NewOAuthUserToken()

		if err := oauthUserTokenService.LoadByID(session, user.UserID, grantID, &oauthUserToken); err != nil {
			return derp.Wrap(err, location, "Loading OAuthUserToken", derp.WithUnauthorized())
		}

		// Call the continuation function
		return fn(ctx, factory, session, &oauthUserToken, user)
	})
}

// WithOwner handles boilerplate code for requests that are restricted to the domain owner
func WithOwner(serverFactory *server.Factory, fn WithFunc0) echo.HandlerFunc {

	const location = "handler.WithAdmin"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

		// Guarantee that the user is signed in
		if authorization := getAuthorization(ctx); !authorization.DomainOwner {
			return derp.Unauthorized(location, "You must be an admin to perform this action")
		}

		// Call the continuation function
		return fn(ctx, factory, session)
	})
}

// WithPrivilege handles boilerplate code for requests that load a Privilege belonging to the requester's Identity
func WithPrivilege(serverFactory *server.Factory, fn WithFunc2[model.Identity, model.Privilege]) echo.HandlerFunc {

	const location = "handler.WithPrivilege"

	return WithIdentity(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, identity *model.Identity) error {

		// Load the Privilege from the database
		privilegeService := factory.Privilege()
		privilege := model.NewPrivilege()

		privilegeID, err := primitive.ObjectIDFromHex(ctx.Param("privilegeId"))

		if err != nil {
			return derp.BadRequest(location, "Invalid PrivilegeID", "PrivilegeID must be a valid ObjectID")
		}

		if err := privilegeService.LoadByIdentity(session, identity.IdentityID, privilegeID, &privilege); err != nil {
			return derp.Wrap(err, location, "Loading Privilege")
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
			return derp.Wrap(err, location, "Loading Product")
		}

		// Load the MerchantAccount used for the Product
		merchantAccountService := factory.MerchantAccount()
		merchantAccount := model.NewMerchantAccount()

		if err := merchantAccountService.LoadByID(session, product.MerchantAccountID, &merchantAccount); err != nil {
			return derp.Wrap(err, location, "Loading MerchantAccount")
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
			return derp.Wrap(err, location, "Loading Registration")
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

		// If we have a valid token, then use it to  look up the search query
		if token := ctx.Param("searchId"); token != "" {
			searchQuery := model.NewSearchQuery()
			if err := searchQueryService.LoadByToken(session, token, &searchQuery); err != nil {
				return derp.Wrap(err, location, "Loading search query from database")
			}

			// Call the continuation function
			return fn(ctx, factory, session, template, stream, &searchQuery)
		}

		// Otherwise, make a new token using the URL parameters provided
		searchQuery, err := searchQueryService.LoadOrCreate(session, ctx.QueryParams())

		if err != nil {
			return derp.Wrap(err, location, "Creating search query token")
		}

		// Call the continuation function
		return fn(ctx, factory, session, template, stream, &searchQuery)
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
		if err := streamService.LoadByToken(session, token, &stream); err != nil {

			// Anything but a "Not Found" error is a problem
			if !derp.IsNotFound(err) {
				return derp.Wrap(err, location, "Loading stream from database")
			}

			// If the "home" page is requested but not found, then we're in "startup" mode
			if token == "home" {
				return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
			}

			// Maybe you want a User, but didn't type the "@" prefix?
			user := model.NewUser()
			if err := factory.User().LoadByUsername(session, token, &user); err == nil {

				// If the user has moved, then forward to the Oracle
				if user.MovedTo != "" {
					ctx.Response().Header().Set(HxRedirectHeader, user.MovedTo)
					return ctx.Redirect(http.StatusSeeOther, user.MovedTo)
				}

				// Forward to the User's page (with the correct URL)
				return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
			}

			// I give up, man..
			return ctx.NoContent(http.StatusNotFound)
		}

		// If this Stream has been moved, then redirect to the Oracle
		if stream.MovedTo != "" {
			newURL := stream.MovedTo + "?url=" + stream.ActivityPubURL()
			ctx.Response().Header().Set(HxRedirectHeader, newURL)
			return ctx.Redirect(http.StatusPermanentRedirect, newURL)
		}

		// Otherwise, continue rendering the Stream
		return fn(ctx, factory, session, &stream)
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

		// Parse/Validate the Username/UserID
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		// Try to load the User from the database
		userService := factory.User()
		user := model.NewUser()
		if err := userService.LoadByToken(session, userID, &user); err != nil {
			return derp.Wrap(err, location, "Loading User")
		}

		// Handle redirects for Users who have moved away.
		if user.MovedTo != "" {
			ctx.Response().Header().Set(HxRedirectHeader, user.MovedTo)
			return ctx.Redirect(http.StatusPermanentRedirect, user.MovedTo)
		}

		// Call the continuation function
		return fn(ctx, factory, session, &user)
	})
}

// WithUserForwarding handles boilerplate code for requests that load a user by username or ID
// and, when called with a UserID/objectId, forwards to the user's correct username
func WithUserForwarding(serverFactory *server.Factory, fn WithFunc1[model.User]) echo.HandlerFunc {

	const location = "handler.WithUserForwarding"

	return WithUser(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

		// Parse/Validate the Username/UserID
		userID, err := profileUsername(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Username")
		}

		// If this is a JSON-LD request, then skip the forwarding and just return the User
		if hannibal.IsActivityPubRequest(ctx.Request()) {
			return activitypub.RenderProfileJSONLD(ctx, factory, session, user)
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
		return fn(ctx, factory, session, user)
	})
}

// WithOAuthUserStream handles boilerplate code for requests that load both a User and a Stream,
// and validates that the Stream is a profile post of the User
func WithOAuthUserStream(serverFactory *server.Factory, fn WithFunc3[model.OAuthUserToken, model.User, model.Stream]) echo.HandlerFunc {

	const location = "handler.WithUser"

	return WithOAuthUser(serverFactory, func(ctx *steranko.Context, factory *service.Factory, session data.Session, oauthUserToken *model.OAuthUserToken, user *model.User) error {

		// Load the Stream from the database
		streamService := factory.Stream()
		stream := model.NewStream()
		streamID, err := primitive.ObjectIDFromHex(ctx.Param("streamId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid StreamID", ctx.Param("streamId"))
		}

		if err := streamService.LoadByID(session, streamID, &stream); err != nil {
			return derp.Wrap(err, location, "Loading Stream", streamID)
		}

		// RULE: Require that this Stream is a part of this User's profile
		if stream.ParentIDs.First() != user.UserID {
			return derp.Forbidden(location, "Stream must be owned by this User")
		}

		// Call the continuation function
		return fn(ctx, factory, session, oauthUserToken, user, &stream)
	})
}
