package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeTiles_CustomURLIsInTheSchema confirms the ZXY Tile URL field and the schema agree.
// They did not until 2026-09-11, and form.Validate is what reports the disagreement.
func TestGeocodeTiles_CustomURLIsInTheSchema(t *testing.T) {

	config := NewGeocodeTiles().ManualConfig()

	require.True(t, indexFormPaths(config)["data.href"], "the form offers a ZXY Tile URL field")

	element, declaredBySchema := config.Schema.GetElement("data.href")
	require.True(t, declaredBySchema)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.False(t, stringElement.Required, "a Domain on a named provider has no custom URL")

	require.NoError(t, config.Validate())
}

// TestGeocodeTiles_CustomURLIsSaved confirms a ZXY template survives a form post, brace
// placeholders included. A "url" format on that property would reject them.
func TestGeocodeTiles_CustomURLIsSaved(t *testing.T) {

	config := NewGeocodeTiles().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.provider": []string{"Custom"},
		"data.style":    []string{"CUSTOM"},
		"data.href":     []string{"https://tile.example.com/{z}/{x}/{y}.png"},
		"active":        []string{"true"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	require.Equal(t, "Custom", data.GetString("provider"))
	require.Equal(t, "CUSTOM", data.GetString("style"))
	require.Equal(t, "https://tile.example.com/{z}/{x}/{y}.png", data.GetString("href"))
}

// TestGeocodeTiles_CustomURLShapesAreSaved walks the template shapes a tile server can need,
// because each carries a character that some schema format would strip.
func TestGeocodeTiles_CustomURLShapesAreSaved(t *testing.T) {

	urls := []string{
		"https://tile.openstreetmap.org/{z}/{x}/{y}.png",
		"https://tile.example.com/{z}/{x}/{y}@2x.png?apikey=abc123",
		"https://{s}.tile.example.com/{z}/{x}/{y}.png",
		"http://192.168.1.10:8080/tiles/{z}/{x}/{y}.png",
	}

	for _, tileURL := range urls {

		t.Run(tileURL, func(t *testing.T) {

			config := NewGeocodeTiles().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider": []string{"Custom"},
				"data.style":    []string{"CUSTOM"},
				"data.href":     []string{tileURL},
			}, testLookupProvider{}))

			require.Equal(t, tileURL, value.GetMap("data").GetString("href"))
		})
	}
}

// TestGeocodeTiles_CustomStyleResolvesTheSavedHref closes the loop through the reader. The
// CUSTOM tile style carries "{href}", which service.GeocodeTiles replaces with data.href.
func TestGeocodeTiles_CustomStyleResolvesTheSavedHref(t *testing.T) {

	var customStyle bool

	for _, lookupCode := range dataset.GeocodeTiles() {
		if lookupCode.Value == "CUSTOM" {
			customStyle = true
			require.Equal(t, "{href}", lookupCode.Href)
		}
	}

	require.True(t, customStyle, "the CUSTOM tile style is still offered")

	config := NewGeocodeTiles().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.provider": []string{"Custom"},
		"data.style":    []string{"CUSTOM"},
		"data.href":     []string{"https://tile.example.com/{z}/{x}/{y}.png"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	require.Equal(t, "CUSTOM", data.GetString("style"), "the style the reader looks up")
	require.NotEmpty(t, data.GetString("href"), "the value the reader substitutes")
}

// TestGeocodeTiles_ProviderHoldsAGroupName pins that the select-group writes a tile GROUP
// name rather than a style value, which is what the show-if comparisons are written against.
func TestGeocodeTiles_ProviderHoldsAGroupName(t *testing.T) {

	groups := make(map[string]bool)

	for _, lookupCode := range dataset.GeocodeTiles() {
		groups[lookupCode.Group] = true
	}

	// Both literals appear in this form's show-if expressions
	assert.True(t, groups["Custom"], "show-if tests data.provider == Custom")
	assert.True(t, groups["Open Street Map"], "show-if tests data.provider != Open Street Map")
}

// TestGeocodeTiles_APIKeyIsHiddenForOpenStreetMap confirms the show-if keeps a posted key out
// of the Data map when the free provider is selected, which needs no credential.
func TestGeocodeTiles_APIKeyIsHiddenForOpenStreetMap(t *testing.T) {

	testCases := map[string]string{
		"Open Street Map": "",
		"Maptiler":        "SECRET-KEY",
		"Custom":          "SECRET-KEY",
	}

	for provider, expectedKey := range testCases {

		t.Run(provider, func(t *testing.T) {

			config := NewGeocodeTiles().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider": []string{provider},
				"data.style":    []string{"MAPTILER-STREETS"},
				"data.apiKey":   []string{"SECRET-KEY"},
			}, testLookupProvider{}))

			require.Equal(t, expectedKey, value.GetMap("data").GetString("apiKey"))
		})
	}
}

// TestGeocodeTiles_LifecycleIsANoOp confirms this provider only supplies a form
func TestGeocodeTiles_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGeocodeTiles().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeTiles().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGeocodeTiles().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeTiles().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}

// TestGeocodeTiles_TypeIsWrittenForTheConnectionLoader confirms the hidden type element
// writes the value that service.Connection.LoadActiveByType queries for.
func TestGeocodeTiles_TypeIsWrittenForTheConnectionLoader(t *testing.T) {

	config := NewGeocodeTiles().ManualConfig()
	require.Equal(t, model.ConnectionTypeGeocodeTiles, hiddenTypeValue(config))

	value := newTestFormValue()
	require.NoError(t, config.SetURLValues(&value, url.Values{
		"type": []string{model.ConnectionTypeGeocodeTiles},
	}, testLookupProvider{}))

	require.Equal(t, model.ConnectionTypeGeocodeTiles, value.GetString("type"))
}
