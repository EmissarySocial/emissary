package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBluesky_BeforeSaveDerivesActive walks every allowType value that can reach
// BeforeSave, including the shapes no form produces.
func TestBluesky_BeforeSaveDerivesActive(t *testing.T) {

	testCases := map[string]struct {
		allowType      string
		expectedActive bool
	}{
		"NONE disables the bridge":          {"NONE", false},
		"GROUPS enables the bridge":         {"GROUPS", true},
		"ALL enables the bridge":            {"ALL", true},
		"an empty value enables it":         {"", true},
		"an unknown value enables it":       {"MAYBE", true},
		"lowercase none enables it":         {"none", true},
		"NONE with whitespace enables it":   {" NONE ", true},
		"an invalid UTF-8 value enables it": {"\xff\xfe", true},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {

			connection := model.NewConnection()
			connection.Data.SetString("allowType", testCase.allowType)

			require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
			require.Equal(t, testCase.expectedActive, connection.Active)
		})
	}
}

// TestBluesky_BeforeSaveOverridesActive confirms BeforeSave always recomputes Active,
// which is why the Bluesky form carries no Enable toggle of its own.
func TestBluesky_BeforeSaveOverridesActive(t *testing.T) {

	testCases := []struct {
		allowType      string
		startActive    bool
		expectedActive bool
	}{
		{"NONE", true, false},
		{"NONE", false, false},
		{"ALL", true, true},
		{"ALL", false, true},
	}

	for _, testCase := range testCases {

		connection := model.NewConnection()
		connection.Active = testCase.startActive
		connection.Data.SetString("allowType", testCase.allowType)

		require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
		assert.Equal(t, testCase.expectedActive, connection.Active, "allowType %q from active=%v", testCase.allowType, testCase.startActive)
	}
}

// TestBluesky_BeforeSaveConvertsNonStringAllowType confirms a non-string value in the Data
// map never panics, and that BeforeSave reads whatever mapof converts it into. A one-element
// slice of "NONE" therefore disables the bridge just as the bare string does.
func TestBluesky_BeforeSaveConvertsNonStringAllowType(t *testing.T) {

	testCases := map[string]struct {
		stored         any
		expectedActive bool
	}{
		"nil":                 {nil, true},
		"an int":              {42, true},
		"a bool":              {true, true},
		"an empty slice":      {[]string{}, true},
		"a map":               {mapof.Any{"allowType": "NONE"}, true},
		"an empty struct":     {struct{}{}, true},
		"a slice of NONE":     {[]string{"NONE"}, false},
		"a slice led by NONE": {[]string{"NONE", "ALL"}, false},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {

			connection := model.NewConnection()
			connection.Data["allowType"] = testCase.stored

			require.NotPanics(t, func() {
				require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
			})

			require.Equal(t, testCase.expectedActive, connection.Active)
		})
	}
}

// TestBluesky_BeforeSaveLeavesTheRestOfTheConnectionAlone confirms Active is the only field written
func TestBluesky_BeforeSaveLeavesTheRestOfTheConnectionAlone(t *testing.T) {

	connection := newTestConnection()
	connection.ProviderID = model.ConnectionProviderBluesky
	connection.Data.SetString("allowType", "NONE")
	connection.Data.SetString("sentinel", "untouched")

	before := connection
	before.Active = false

	require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
	require.Equal(t, before, connection)
}

/******************************************
 * Settings Form
 ******************************************/

// TestBluesky_ManualConfigSchema confirms the schema matches what model.Domain reads back
func TestBluesky_ManualConfigSchema(t *testing.T) {

	config := NewBluesky().ManualConfig()

	allowType, exists := config.Schema.GetElement("data.allowType")
	require.True(t, exists)

	allowTypeString, isString := allowType.(schema.String)
	require.True(t, isString)
	require.True(t, allowTypeString.Required)
	require.Equal(t, []string{"ALL", "GROUPS", "NONE"}, allowTypeString.Enum)

	shareGroups, exists := config.Schema.GetElement("data.shareGroups")
	require.True(t, exists)
	require.IsType(t, schema.Array{}, shareGroups)
}

// TestBluesky_ManualConfigHasNoActiveToggle confirms the form never writes Active directly,
// because BeforeSave derives it from allowType.
func TestBluesky_ManualConfigHasNoActiveToggle(t *testing.T) {

	config := NewBluesky().ManualConfig()

	_, exists := config.Schema.GetElement("active")
	require.True(t, exists, "the Connection object still carries an active field")

	for _, element := range config.Element.AllElements() {
		require.NotEqual(t, "active", element.Path, "the form must not offer an Enable toggle")
	}
}

// TestBluesky_ShareGroupsPostedAsAPointer pins the shape that a form post leaves in the Data
// map: schemaSafeValue wraps an Array path in a *sliceof.String, and GetSliceOfString reads it.
func TestBluesky_ShareGroupsPostedAsAPointer(t *testing.T) {

	config := NewBluesky().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.allowType":   []string{"GROUPS"},
		"data.shareGroups": []string{"5f8e1d2c3b4a5968770a1b2c", "5f8e1d2c3b4a5968770a1b2d"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	require.IsType(t, &sliceof.String{}, data["shareGroups"])
	require.Equal(t, []string{"5f8e1d2c3b4a5968770a1b2c", "5f8e1d2c3b4a5968770a1b2d"}, data.GetSliceOfString("shareGroups"))
}

// TestBluesky_ShareGroupsOnlyWrittenWhenGroupsSelected confirms the multiselect's show-if
// keeps a posted group list out of the Data map unless allowType is GROUPS.
func TestBluesky_ShareGroupsOnlyWrittenWhenGroupsSelected(t *testing.T) {

	for _, allowType := range []string{"ALL", "NONE", ""} {

		t.Run("allowType="+allowType, func(t *testing.T) {

			config := NewBluesky().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.allowType":   []string{allowType},
				"data.shareGroups": []string{"5f8e1d2c3b4a5968770a1b2c"},
			}, testLookupProvider{}))

			require.Empty(t, value.GetMap("data").GetSliceOfString("shareGroups"))
		})
	}
}

// TestBluesky_DisablingTheBridgeKeepsTheGroupList pins that switching allowType to NONE
// leaves the previously selected groups in place, ready to return if it is switched back.
func TestBluesky_DisablingTheBridgeKeepsTheGroupList(t *testing.T) {

	config := NewBluesky().ManualConfig()

	value := mapof.Any{
		"active": true,
		"data":   mapof.Any{"allowType": "GROUPS", "shareGroups": []string{"5f8e1d2c3b4a5968770a1b2c"}},
	}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.allowType": []string{"NONE"},
	}, testLookupProvider{}))

	require.Equal(t, "NONE", value.GetMap("data").GetString("allowType"))
	require.Equal(t, []string{"5f8e1d2c3b4a5968770a1b2c"}, value.GetMap("data").GetSliceOfString("shareGroups"))
}
