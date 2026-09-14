package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/derp/plugins"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// capturingReporter keeps the errors that derp.Report fans out, so a test can inspect
// exactly what would have reached the console and the ErrorLog collection
type capturingReporter struct {
	reported []error
}

func (reporter *capturingReporter) Report(err error) {
	reporter.reported = append(reporter.reported, err)
}

// TestErrorHandler_RedactsCredentials confirms the header block in an error report
// carries no credential
func TestErrorHandler_RedactsCredentials(t *testing.T) {

	// errorHandler reports the inbound header block as a derp detail. derp stores
	// details verbatim and fans them out to the console AND the durable ErrorLog, so a
	// raw header block publishes the caller's cookie and bearer token. (BUG-64)

	reporter := &capturingReporter{}
	derp.SetPlugins(reporter)
	defer derp.SetPlugins(plugins.JSON{}) // derp's own init() default

	request := httptest.NewRequest(http.MethodGet, "https://example.com/missing", nil)
	request.Header.Set("Authorization", "Bearer super-secret-token")
	request.Header.Set("Cookie", "session=super-secret-cookie")
	request.Header.Set("Signature", `keyId="x",signature="super-secret-signature"`)
	request.Header.Set("Accept", "text/html")

	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(request, recorder)

	// RULE: a 500, not a 401. Every IsUnauthorized branch returns before derp.Report,
	// so a 401 would pass this test without exercising the reporting path at all.
	errorHandler(derp.InternalError("test", "Something broke"), ctx)

	require.Len(t, reporter.reported, 1, "the error must actually have been reported")

	serialized, marshalError := json.Marshal(reporter.reported[0])

	require.NoError(t, marshalError)
	require.NotContains(t, string(serialized), "super-secret", "no credential may reach a log")
	require.Contains(t, string(serialized), "text/html", "harmless headers are still worth reporting")
}

// TestErrorHandler_DoesNotMutateTheRequest guards the copy that redaction depends on
func TestErrorHandler_DoesNotMutateTheRequest(t *testing.T) {

	// Redacting in place would strip the credential from the live request, which the
	// rest of the response cycle is still using.
	reporter := &capturingReporter{}
	derp.SetPlugins(reporter)
	defer derp.SetPlugins(plugins.JSON{}) // derp's own init() default

	request := httptest.NewRequest(http.MethodGet, "https://example.com/missing", nil)
	request.Header.Set("Authorization", "Bearer still-needed")

	ctx := echo.New().NewContext(request, httptest.NewRecorder())

	errorHandler(derp.InternalError("test", "Something broke"), ctx)

	require.Equal(t, "Bearer still-needed", request.Header.Get("Authorization"))
}
