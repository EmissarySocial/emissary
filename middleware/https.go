package middleware

import (
	"net/http"

	"github.com/benpate/uri"
	"github.com/labstack/echo/v4"
)

// HttpsRedirect enforces HTTPS for public traffic: insecure requests are redirected
// to their HTTPS URL, and secure requests are stamped with an HSTS header so the
// browser upgrades every future request itself. Local and private-network hosts are
// exempt from both, because Emissary serves them over plain HTTP.
func HttpsRedirect(handler echo.HandlerFunc) echo.HandlerFunc {

	return func(context echo.Context) error {

		// Two-year max-age, without includeSubDomains or preload: Emissary is
		// multi-domain and often proxied, so those (hard-to-reverse) options are
		// left as a deliberate operator choice rather than a default.
		const hstsHeaderValue = "max-age=63072000"

		request := context.Request()

		// Local domains are served over plain HTTP by design, so neither the
		// redirect nor the HSTS policy applies. IsLocalHostname also covers any
		// non-public IP, so private-network deployments are exempt too.
		if uri.IsLocalHostname(request.Host) {
			return handler(context)
		}

		// A secure public request: assert the HSTS policy, then continue.
		if context.Scheme() == "https" {
			context.Response().Header().Set("Strict-Transport-Security", hstsHeaderValue)
			return handler(context)
		}

		// An insecure public request: permanently redirect to the HTTPS URL.
		request.URL.Scheme = "https"

		return context.Redirect(http.StatusPermanentRedirect, request.URL.String())
	}
}
