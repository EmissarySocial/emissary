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

// TestUnsplash_AllFourCredentialsAreRequired pins the credential set, because Unsplash needs an
// application identity alongside the keys.
func TestUnsplash_AllFourCredentialsAreRequired(t *testing.T) {

	config := NewUnsplash().ManualConfig()

	for _, path := range []string{
		"data.applicationId",
		"data.applicationName",
		"data.accessKey",
		"data.secretKey",
	} {
		element, exists := config.Schema.GetElement(path)
		require.True(t, exists, "missing %s", path)

		stringElement, isString := element.(schema.String)
		require.True(t, isString)
		assert.True(t, stringElement.Required, "%s must be required", path)
	}
}

// TestUnsplash_SecretKeyIsStoredInDataNotTheVault pins that both Unsplash keys land in data
// rather than the encrypted vault, unlike the payment providers' credentials.
func TestUnsplash_SecretKeyIsStoredInDataNotTheVault(t *testing.T) {

	config := NewUnsplash().ManualConfig()

	for _, path := range []string{"vault.secretKey", "vault.accessKey"} {
		_, inVault := config.Schema.GetElement(path)
		assert.False(t, inVault, "%s is not where this form writes", path)
	}
}

// TestUnsplash_TypeIsImage confirms the hidden type element writes the image connection type
func TestUnsplash_TypeIsImage(t *testing.T) {

	config := NewUnsplash().ManualConfig()
	require.Equal(t, model.ConnectionTypeImage, hiddenTypeValue(config))
	require.Equal(t, "IMAGE", model.ConnectionTypeImage)
}

// TestUnsplash_SavesAFullConfiguration confirms every field round-trips
func TestUnsplash_SavesAFullConfiguration(t *testing.T) {

	config := NewUnsplash().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"type":                 []string{model.ConnectionTypeImage},
		"data.applicationId":   []string{"12345"},
		"data.applicationName": []string{"Emissary"},
		"data.accessKey":       []string{"access-key"},
		"data.secretKey":       []string{"secret-key"},
		"active":               []string{"true"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	assert.Equal(t, model.ConnectionTypeImage, value.GetString("type"))
	assert.Equal(t, "12345", data.GetString("applicationId"))
	assert.Equal(t, "Emissary", data.GetString("applicationName"))
	assert.Equal(t, "access-key", data.GetString("accessKey"))
	assert.Equal(t, "secret-key", data.GetString("secretKey"))
	assert.True(t, value.GetBool("active"))
}

// TestUnsplash_NoFieldIsGated confirms no Unsplash field carries a show-if, so every field is
// written on every post.
func TestUnsplash_NoFieldIsGated(t *testing.T) {

	config := NewUnsplash().ManualConfig()

	for _, element := range config.Element.AllElements() {
		assert.Empty(t, element.Options.GetString("show-if"), "%s is ungated", element.Path)
	}
}

// TestUnsplash_RequiredCredentialsSurviveAPartialPost pins that all four credentials are
// Required, which is what refuses the empty write a partial post attempts.
func TestUnsplash_RequiredCredentialsSurviveAPartialPost(t *testing.T) {

	config := NewUnsplash().ManualConfig()

	value := mapof.Any{
		"type":   model.ConnectionTypeImage,
		"active": true,
		"data":   mapof.Any{"accessKey": "EXISTING-KEY", "secretKey": "EXISTING-SECRET"},
	}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.applicationId": []string{"12345"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	assert.Equal(t, "12345", data.GetString("applicationId"), "the posted field is written")
	assert.Equal(t, "EXISTING-KEY", data.GetString("accessKey"))
	assert.Equal(t, "EXISTING-SECRET", data.GetString("secretKey"))
}

// TestUnsplash_LifecycleIsANoOp confirms this provider only supplies a form
func TestUnsplash_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewUnsplash().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewUnsplash().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewUnsplash().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewUnsplash().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
