package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetTranslationLanguages answers Mastodon's translation language list with an empty list
func GetTranslationLanguages(ctx echo.Context) error {

	// Emissary offers no translation (the instance reports it disabled), and clients read an empty
	// list as "nothing to translate"
	return ctx.JSON(http.StatusOK, map[string][]string{})
}
