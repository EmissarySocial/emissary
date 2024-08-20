package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// WithFactory handles boilerplate code for requests that require only the domain Factory
func WithFactory(serverFactory *server.Factory, fn func(ctx *steranko.Context, factory *domain.Factory) error) func(ctx echo.Context) error {

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

// WithStream handles boilerplate code for requests that load a stream
func WithStream(serverFactory *server.Factory, fn func(ctx *steranko.Context, factory *domain.Factory, stream *model.Stream) error) func(ctx echo.Context) error {

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

		// Load the Stream from the database
		streamService := factory.Stream()
		stream := model.NewStream()
		token := ctx.Param("stream")

		if err := streamService.LoadByToken(token, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading stream from database")
		}

		// Call the continuation function
		return fn(sterankoContext, factory, &stream)
	}
}

// WithUser handles boilerplate code for requests that load a user by username or ID
func WithUser(serverFactory *server.Factory, fn func(ctx *steranko.Context, factory *domain.Factory, user *model.User) error) func(ctx echo.Context) error {

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

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userID := ctx.Param("user")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user from database")
		}

		// Call the continuation function
		return fn(sterankoContext, factory, &user)
	}
}

// WithAuthenticatedUser handles boilerplate code for requests that require a signed-in user
func WithAuthenticatedUser(serverFactory *server.Factory, fn func(ctx *steranko.Context, factory *domain.Factory, user *model.User) error) func(ctx echo.Context) error {

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
		return fn(sterankoContext, factory, &user)
	}
}
