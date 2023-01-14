package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/labstack/echo/v4"
)

// TODO: MEDIUM: Enable Remote Following Server:
// https://www.hughrundle.net/how-to-implement-remote-following-for-your-activitypub-project/
// oStatus Discovery Docs: http://ostatus.github.io/spec/OStatus%201.0%20Draft%202.html#anchor10

func GetOStatusSubscribe(serverFactory *server.Factory) echo.HandlerFunc {
	return func(context echo.Context) error {
		return nil
	}
}

// TODO: MEDIUM: Enable Remote Following Client:
// Add a "follow" button to user's profiles that does autodiscovery for remote follows
/// (or adds an email subscription?) (or gives a signup link?)
