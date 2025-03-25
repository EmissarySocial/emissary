package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *domain.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

// canInfo returns TRUE if zerolog is configured to allow Info logs
// nolint:unused
func canInfo() bool {
	return canLog(zerolog.InfoLevel)
}

// canDebug returns TRUE if zerolog is configured to allow Debug logs
// nolint:unused
func canDebug() bool {
	return canLog(zerolog.DebugLevel)
}

// canTrace returns TRUE if zerolog is configured to allow Trace logs
// nolint:unused
func canTrace() bool {
	return canLog(zerolog.TraceLevel)
}

// canLog is a silly zerolog helper that returns TRUE
// if the provided log level would be allowed
// (based on the global log level).
// This makes it easier to execute expensive code conditionally,
// for instance: marshalling a JSON object for logging.
func canLog(level zerolog.Level) bool {
	return zerolog.GlobalLevel() <= level
}
