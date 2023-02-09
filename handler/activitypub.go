package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func ActivityPub_PostInbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_PostInbox"

	return func(ctx echo.Context) error {

		// Find the factory for this hostname
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error creating ActivityStreamsHandler"))
		}

		// Create a new ActivityPub Actor
		actor := factory.ActivityPub_Actor()

		// Try to handle the ActivityPub request
		isActivityPubRequest, err := actor.PostInbox(ctx.Request().Context(), ctx.Response().Writer, ctx.Request())

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error creating ActivityStreamsHandler"))
		} else if isActivityPubRequest {
			return nil
		}

		// Otherwise, this is not an ActivityPub request
		return derp.NewBadRequestError(location, "Not an ActivityPub request")
	}
}

func ActivityPub_GetOutbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetOutbox"

	return func(ctx echo.Context) error {

		// Find the factory for this hostname
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error creating ActivityStreamsHandler"))
		}

		// Create a new ActivityPub Actor
		actor := factory.ActivityPub_Actor()

		// Try to handle the ActivityPub request
		isActivityPubRequest, err := actor.GetOutbox(ctx.Request().Context(), ctx.Response().Writer, ctx.Request())

		if err != nil {
			return derp.Report(derp.Wrap(err, location, "Error creating ActivityStreamsHandler"))
		} else if isActivityPubRequest {
			return nil
		}

		// Otherwise, this is not an ActivityPub request
		return derp.NewBadRequestError(location, "Not an ActivityPub request")
	}
}

func ActivityPub_GetOutboxItem(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetOutboxItem"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}

func ActivityPub_GetPublicKey(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetPublicKey"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}

func ActivityPub_GetBlocks(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetBlocks"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}

func ActivityPub_GetFollowers(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetFollowers"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}

func ActivityPub_GetFollowing(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetFollowing"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}

func ActivityPub_GetLikes(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetLikes"

	return func(ctx echo.Context) error {
		// TODO: CRITICAL: Implement this
		return nil
	}
}
