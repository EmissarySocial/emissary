package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestRegistrationStripe_PortalURLIsAWebLink confirms that the Stripe registration's
// customer portal URL accepts only an http or https link (BUG-236)
func TestRegistrationStripe_PortalURLIsAWebLink(t *testing.T) {

	definition, err := os.ReadFile("../_embed/templates/registration-stripe/registration.hjson")
	require.NoError(t, err)

	registration := model.NewRegistration("register-stripe", testFuncMap())
	require.NoError(t, hjson.Unmarshal(definition, &registration))

	// The placeholder shown in the form is a valid value
	data := mapof.NewString()
	require.NoError(t, registration.Schema.Set(&data, "stripe_portal_url", "https://billing.stripe.com/p/login/prod_xxxxxxxxx"))

	// Opaque schemes are rejected, even though each one is a valid URI
	for _, hostile := range []string{"javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "mailto:billing@example.com"} {
		data := mapof.NewString()
		require.Error(t, registration.Schema.Set(&data, "stripe_portal_url", hostile), hostile)
	}
}
