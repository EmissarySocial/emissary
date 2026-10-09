package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStripe verifies that the Stripe provider satisfies the ManualProvider interface
func TestStripe(t *testing.T) {

	var stripe any = NewStripe()
	provider := stripe.(ManualProvider)

	require.NotNil(t, provider)
}

// TestStripe_IsNotReachableFromTheProviderRegistry pins Stripe as dormant code. Its
// ConnectionProvider constant is commented out of the Connection schema's providerId enum,
// so service.Provider.GetProvider can never return it.
func TestStripe_IsNotReachableFromTheProviderRegistry(t *testing.T) {

	schemaElement, isObject := model.ConnectionSchema().(schema.Object)
	require.True(t, isObject)

	providerID, isString := schemaElement.Properties["providerId"].(schema.String)
	require.True(t, isString)

	assert.NotContains(t, providerID.Enum, "STRIPE")
	assert.Contains(t, providerID.Enum, model.ConnectionProviderStripeConnect, "Stripe Connect is the live payment provider")
}

// TestStripe_ManualConfigDeclaresAKeyItNeverOffers pins a gap in the settings form: the schema
// requires data.apiKey, but the form has no element for it, so it can never be filled in.
func TestStripe_ManualConfigDeclaresAKeyItNeverOffers(t *testing.T) {

	config := NewStripe().ManualConfig()

	element, exists := config.Schema.GetElement("data.apiKey")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.True(t, stringElement.Required)

	require.False(t, indexFormPaths(config)["data.apiKey"], "no form element writes the API key")
}

// TestStripe_TypeIsUserPayment confirms the hidden type element writes the payment type
func TestStripe_TypeIsUserPayment(t *testing.T) {

	config := NewStripe().ManualConfig()
	require.Equal(t, model.ConnectionTypeUserPayment, hiddenTypeValue(config))
	require.Equal(t, "USER-PAYMENT", model.ConnectionTypeUserPayment)
}

// TestStripe_SavesOnlyTypeAndActive confirms the form writes its two elements and nothing else
func TestStripe_SavesOnlyTypeAndActive(t *testing.T) {

	config := NewStripe().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"type":        []string{model.ConnectionTypeUserPayment},
		"data.apiKey": []string{"sk_test_ignored"},
		"active":      []string{"true"},
	}, testLookupProvider{}))

	require.Equal(t, model.ConnectionTypeUserPayment, value.GetString("type"))
	require.True(t, value.GetBool("active"))
	require.Empty(t, value.GetMap("data").GetString("apiKey"), "a path with no form element is never written")
}

// TestStripe_LifecycleIsANoOp confirms all four lifecycle methods write nothing
func TestStripe_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewStripe().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewStripe().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewStripe().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewStripe().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
