package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// GetHostMeta serves the /.well-known/host-meta document, in XRD or JSON format
func GetHostMeta(serverFactory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		// Use JSON respone if requested
		if ctx.Request().Header.Get("Accept") == "application/json" {
			return GetHostMetaJSON(serverFactory)(ctx)
		}

		// Generate XML/XRD response
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetHostMeta", "Invalid domain name.")
		}

		result := `
		<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
			<Link rel="lrdd" template="` + factory.Host() + `/.well-known/webfinger?resource={uri}"/>
		</XRD>`

		result = strings.ReplaceAll(result, "\t", "")
		resultBytes := []byte(result)

		response := ctx.Response()
		response.Header().Set("Content-Type", "application/xrd+xml")
		_, err = response.Write(resultBytes)
		return err
	}
}

// GetHostMetaJSON serves the /.well-known/host-meta document in JSON format
func GetHostMetaJSON(serverFactory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetHostMetaJSON", "Invalid domain name.")
		}

		result := map[string]any{
			"links": []map[string]string{
				{
					"rel":      "lrdd",
					"template": factory.Host() + "/.well-known/webfinger?resource={uri}",
				},
			},
		}

		return ctx.JSON(200, result)
	}
}

// GetChangePassword redirects the legacy password-change URL to the password reset page
func GetChangePassword(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusTemporaryRedirect, "/signin/reset")
	}
}
