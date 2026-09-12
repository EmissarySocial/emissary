package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeTiles_CustomURLIsNotInTheSchema pins the defect that form.Validate reports: the
// ZXY Tile URL field writes data.href, which the schema never declares.
func TestGeocodeTiles_CustomURLIsNotInTheSchema(t *testing.T) {

	config := NewGeocodeTiles().ManualConfig()

	_, declaredByTheForm := indexFormPaths(config)["data.href"]
	require.True(t, declaredByTheForm, "the form still offers a ZXY Tile URL field")

	_, declaredBySchema := config.Schema.GetElement("data.href")
	require.False(t, declaredBySchema, "the schema still omits data.href")

	require.Error(t, config.Validate(), "form.Validate is what catches this")
}

// TestGeocodeTiles_CustomURLIsSilentlyDropped pins the consequence. An admin fills in the
// ZXY Tile URL, the save reports success, and the value never reaches the Connection.
func TestGeocodeTiles_CustomURLIsSilentlyDropped(t *testing.T) {

	config := NewGeocodeTiles().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.provider": []string{"Custom"},
		"data.style":    []string{"CUSTOM"},
		"data.href":     []string{"https://tile.example.com/{z}/{x}/{y}.png"},
		"active":        []string{"true"},
	}, testLookupProvider{}), "a rejected path is logged, not returned")

	data := value.GetMap("data")

	require.Equal(t, "Custom", data.GetString("provider"), "the rest of the form saves normally")
	require.Equal(t, "CUSTOM", data.GetString("style"))
	require.Empty(t, data.GetString("href"), "the custom tile URL is gone")
}

// TestGeocodeTiles_CustomStyleNeedsTheDroppedHref ties the dropped field to the feature it
// breaks: the CUSTOM tile style substitutes data.href, so it resolves to an empty URL.
func TestGeocodeTiles_CustomStyleNeedsTheDroppedHref(t *testing.T) {

	var customStyle bool

	for _, lookupCode := range dataset.GeocodeTiles() {
		if lookupCode.Value == "CUSTOM" {
			customStyle = true
			require.Equal(t, "{href}", lookupCode.Href, "service.GeocodeTiles replaces this with data.href")
		}
	}

	require.True(t, customStyle, "the CUSTOM tile style is still offered")
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
