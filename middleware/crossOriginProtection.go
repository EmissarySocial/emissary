package middleware

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// CrossOriginProtection returns a middleware that rejects state-changing requests from other
// web origins, except on the open routes, which are ServeMux patterns such as "POST /oauth/token".
func CrossOriginProtection(openRoutes ...string) echo.MiddlewareFunc {

	// The standard library implementation inspects the Sec-Fetch-Site and Origin headers,
	// so browser-based cross-site requests are refused, while safe methods (GET/HEAD/OPTIONS)
	// and server-to-server requests (ActivityPub, webhooks) — which carry neither header —
	// pass through unchanged.
	protection := http.NewCrossOriginProtection()

	// RULE: An open route must authorize by a value in the request, never by a cookie
	for _, openRoute := range openRoutes {
		protection.AddInsecureBypassPattern(openRoute)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			// RULE: Cross-origin browser requests may not change server state
			if err := protection.Check(ctx.Request()); err != nil {
				return derp.Forbidden("middleware.CrossOriginProtection", "Cross-origin request blocked", err.Error())
			}

			// You may pass.
			return next(ctx)
		}
	}
}
