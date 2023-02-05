package handler

import (
	"time"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func Inbox_MarkRead(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.Inbox_MarkRead"

	return func(ctx echo.Context) error {

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot find Domain")
		}

		// Get the UserID from the URL (could be "me")
		userID, err := authenticatedID(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error retreiving user ID")
		}

		// Try to load the inboxItem from the database
		inboxService := factory.Inbox()

		return inboxService.SetReadDate(userID, ctx.Param("item"), time.Now().Unix())
	}
}

func Inbox_MarkUnRead(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.Inbox_MarkRead"

	return func(ctx echo.Context) error {

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot find Domain")
		}

		// Get the UserID from the URL (could be "me")
		userID, err := authenticatedID(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error retreiving user ID")
		}

		// Try to load the inboxItem from the database
		inboxService := factory.Inbox()

		return inboxService.SetReadDate(userID, ctx.Param("item"), 0)
	}
}
