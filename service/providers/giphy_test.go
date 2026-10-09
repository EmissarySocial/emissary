package providers

import (
	"net/url"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestGiphy_APIKeyConstantMatchesTheSchemaPath confirms the exported vault key and the schema
// property are the same string, because a reader uses the constant and the form uses the path.
func TestGiphy_APIKeyConstantMatchesTheSchemaPath(t *testing.T) {

	require.Equal(t, "apiKey", Giphy_APIKey)

	config := NewGiphy().ManualConfig()
	_, exists := config.Schema.GetElement("data." + Giphy_APIKey)
	require.True(t, exists)
}

// TestGiphy_APIKeyIsStoredInDataNotTheVault pins where the credential lands. Unlike the payment
// providers, Giphy's key is written to data, so it is not encrypted at rest.
func TestGiphy_APIKeyIsStoredInDataNotTheVault(t *testing.T) {

	config := NewGiphy().ManualConfig()

	element, exists := config.Schema.GetElement("data.apiKey")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.True(t, stringElement.Required)

	_, inVault := config.Schema.GetElement("vault.apiKey")
	require.False(t, inVault)
}

// TestGiphy_ManualConfigWritesNoType pins that this form leaves Type empty, so a Giphy
// Connection is found by its providerId rather than by type.
func TestGiphy_ManualConfigWritesNoType(t *testing.T) {

	config := NewGiphy().ManualConfig()

	require.Empty(t, hiddenTypeValue(config))

	_, exists := config.Schema.GetElement("type")
	require.False(t, exists, "the schema does not declare a type either")
}

// TestGiphy_SavesAFullConfiguration confirms the two-field form round-trips
func TestGiphy_SavesAFullConfiguration(t *testing.T) {

	config := NewGiphy().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.apiKey": []string{"giphy-api-key"},
		"active":      []string{"true"},
	}, testLookupProvider{}))

	require.Equal(t, "giphy-api-key", value.GetMap("data").GetString("apiKey"))
	require.True(t, value.GetBool("active"))
}

// TestGiphy_RequiredAPIKeySurvivesAPartialPost pins what Required buys here. The key field
// carries no show-if, so a post that omits it still tries to write an empty value, and the
// schema's Required rule is the only thing that refuses and leaves the stored key in place.
func TestGiphy_RequiredAPIKeySurvivesAPartialPost(t *testing.T) {

	config := NewGiphy().ManualConfig()
	value := mapof.Any{"active": true, "data": mapof.Any{"apiKey": "EXISTING-KEY"}}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"active": []string{"true"},
	}, testLookupProvider{}))

	require.Equal(t, "EXISTING-KEY", value.GetMap("data").GetString("apiKey"))
}

// TestGiphy_APIKeyCannotBeClearedThroughTheForm pins the other side of that rule: because the
// field is Required, an explicitly empty value is refused too, so the form cannot remove a key.
func TestGiphy_APIKeyCannotBeClearedThroughTheForm(t *testing.T) {

	config := NewGiphy().ManualConfig()
	value := mapof.Any{"active": true, "data": mapof.Any{"apiKey": "EXISTING-KEY"}}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.apiKey": []string{""},
		"active":      []string{"true"},
	}, testLookupProvider{}))

	require.Equal(t, "EXISTING-KEY", value.GetMap("data").GetString("apiKey"))
}

// TestGiphy_LifecycleIsANoOp confirms this provider only supplies a form
func TestGiphy_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGiphy().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGiphy().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGiphy().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGiphy().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
