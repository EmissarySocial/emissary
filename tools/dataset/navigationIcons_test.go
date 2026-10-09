package dataset_test

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

// vendoredIconsFile is the icon set that the theme actually serves.
const vendoredIconsFile = "../../_embed/templates/theme-global/resources/bootstrap-icons-1.13.1/bootstrap-icons.json"

// glyphPattern pulls the bootstrap class name back out of the icon service's markup.
var glyphPattern = regexp.MustCompile(`bi bi-([a-z0-9-]+)`)

// hasBothVariants reports whether navigation templates draw a distinct, existing
// filled glyph for this value, matching {{icon $icon}} and {{icon (join $icon "-fill")}}.
func hasBothVariants(icons service.Icons, available map[string]any, value string) bool {

	regular := glyphPattern.FindStringSubmatch(icons.Get(value))
	filled := glyphPattern.FindStringSubmatch(icons.Get(value + "-fill"))

	if len(regular) != 2 || len(filled) != 2 {
		return false
	}

	if _, exists := available[regular[1]]; !exists {
		return false
	}

	if _, exists := available[filled[1]]; !exists {
		return false
	}

	// The filled glyph must be the same shape, not a different icon
	return filled[1] == regular[1]+"-fill"
}

// vendoredIcons reads the names the vendored Bootstrap Icons set defines.
func vendoredIcons(t *testing.T) map[string]any {
	t.Helper()

	file, err := os.ReadFile(vendoredIconsFile)
	require.NoError(t, err, "the vendored icon set must be readable")

	result := make(map[string]any)
	require.NoError(t, json.Unmarshal(file, &result))

	return result
}

func TestNavigationIcons_EveryIconHasBothVariants(t *testing.T) {

	icons := service.NewIcons()
	available := vendoredIcons(t)

	for _, lookupCode := range dataset.NavigationIcons() {
		require.True(t, hasBothVariants(icons, available, lookupCode.Value),
			"value %q lacks a matching filled variant", lookupCode.Value)
	}
}

func TestNavigationIcons_IsTheCompleteSubsetOfIcons(t *testing.T) {

	icons := service.NewIcons()
	available := vendoredIcons(t)

	// Every qualifying icon, in the same order and with the same fields
	expected := make([]string, 0)
	for _, lookupCode := range dataset.Icons() {
		if hasBothVariants(icons, available, lookupCode.Value) {
			expected = append(expected, lookupCode.Value)
		}
	}

	actual := make([]string, 0)
	for _, lookupCode := range dataset.NavigationIcons() {
		actual = append(actual, lookupCode.Value)
	}

	require.Equal(t, expected, actual, "NavigationIcons must be every icon in Icons() that has both variants")
}

func TestNavigationIcons_MatchTheirEntryInIcons(t *testing.T) {

	all := make(map[string]form.LookupCode)
	for _, lookupCode := range dataset.Icons() {
		all[lookupCode.Value] = lookupCode
	}

	for _, lookupCode := range dataset.NavigationIcons() {
		original, exists := all[lookupCode.Value]
		require.True(t, exists, "value %q is not in Icons()", lookupCode.Value)
		require.Equal(t, original, lookupCode, "value %q differs from its entry in Icons()", lookupCode.Value)
	}
}

func TestNavigationIcons_CallersCannotCorruptTheSharedList(t *testing.T) {

	first := dataset.NavigationIcons()
	require.NotEmpty(t, first)

	original := first[0]
	slices.Reverse(first)

	second := dataset.NavigationIcons()
	require.NotEmpty(t, second)
	require.Equal(t, original, second[0], "NavigationIcons() handed out its own backing array")
}
