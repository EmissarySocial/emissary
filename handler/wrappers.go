package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
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
			return derp.NewInternalError(location, "Context must be a Steranko Context")
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

// WithDomain handles boilerplate code for requests that load a domain object
func WithDomain(serverFactory *server.Factory, fn WithFunc1[model.Domain]) echo.HandlerFunc {

	const location = "handler.WithRegistration"

	return WithFactory(serverFactory, func(ctx *steranko.Context, factory *domain.Factory) error {

		// Try to retrieve the Domain record
		domainService := factory.Domain()
		domain, err := domainService.LoadDomain()

		if err != nil {
			return derp.Wrap(err, location, "Unable to load Domain")
		}

		return fn(ctx, factory, &domain)
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
			return derp.Wrap(err, location, "Error loading registration")
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
		searchQuery := model.NewSearchQuery()
		token := ctx.Param("searchId")

		if err := searchQueryService.LoadByToken(token, &searchQuery); err != nil {
			return derp.Wrap(err, location, "Error loading search query from database")
		}

		// Call the continuation function
		return fn(ctx, factory, template, stream, &searchQuery)
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
			if derp.NotFound(err) && (token == "home") {
				return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
			}

			return derp.Wrap(err, location, "Error loading stream from database")
		}

		// Call the continuation function
		return fn(ctx, factory, &stream)
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
		userID := ctx.Param("user")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user from database")
		}

		// Call the continuation function
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
			return derp.NewUnauthorizedError(location, "You must be signed in to perform this action")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user from database")
		}

		// Call the continuation function
		return fn(ctx, factory, &user)
	})
}
