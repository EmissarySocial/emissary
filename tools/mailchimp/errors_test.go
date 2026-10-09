package mailchimp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestDescribeError_SurvivesWrapping guards the reason a User reads on the settings form
func TestDescribeError_SurvivesWrapping(t *testing.T) {

	// build.inlineErrorMessage reads the ROOT message of a 422 chain and the OUTERMOST
	// message of anything else. These errors travel up through a service and a pipeline
	// step, each of which wraps -- so the sentence has to sit at the root to survive.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	err := testClient(server.URL).Ping()

	require.Error(t, err)

	wrapped := derp.Wrap(err, "service.UserConnection.Save", "Unable to connect to Mailchimp")

	require.True(t, derp.IsValidationError(wrapped), "a fixable input error must read as a validation error")
	require.Contains(t, derp.RootMessage(wrapped), "API key", "the reason must outlive its wrappers")
}

// TestDescribeError_TransientFailuresAreNotValidation confirms a failure the User cannot
// fix does not pose as a form error
func TestDescribeError_TransientFailuresAreNotValidation(t *testing.T) {

	// A 500 from Mailchimp is not the User's input being wrong, so it keeps its cause and
	// reads as a generic failure rather than as an instruction they cannot act on.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	err := testClient(server.URL).Ping()

	require.Error(t, err)
	require.False(t, derp.IsValidationError(err))
}

// TestValidateAPIKey_MessageIsTheRoot confirms an entry check reads through a wrapper
func TestValidateAPIKey_MessageIsTheRoot(t *testing.T) {

	wrapped := derp.Wrap(ValidateAPIKey(""), "service.UserConnection.Save", "Unable to connect to Mailchimp")

	require.True(t, derp.IsValidationError(wrapped))
	require.Contains(t, derp.RootMessage(wrapped), "required")
}
