package dataset

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/benpate/icon/bootstrap"
	"github.com/stretchr/testify/require"
)

// bootstrapIconsFile is the vendored icon set that the theme actually serves.
const bootstrapIconsFile = "../../_embed/templates/theme-global/resources/bootstrap-icons-1.13.1/bootstrap-icons.json"

// legacyIconValues are the values that shipped before the set was expanded, and
// that Domain records may already hold. Dropping one orphans stored data.
var legacyIconValues = []string{
	"archive", "book", "bookmark", "box", "calendar", "circle", "clipboard",
	"clock", "cloud", "database", "email", "file", "flag", "folder", "globe",
	"heart", "home", "info", "lock", "rss", "shield", "star", "person", "people",
}

// iconClassPattern pulls the bootstrap class name back out of the provider's markup.
var iconClassPattern = regexp.MustCompile(`bi bi-([a-z0-9-]+)`)

// availableIcons reads the names the vendored Bootstrap Icons stylesheet defines.
func availableIcons(t *testing.T) map[string]any {
	t.Helper()

	file, err := os.ReadFile(bootstrapIconsFile)
	require.NoError(t, err, "the vendored icon set must be readable")

	result := make(map[string]any)
	require.NoError(t, json.Unmarshal(file, &result))

	return result
}

func TestIcons_EveryLegacyValueSurvives(t *testing.T) {

	values := make(map[string]bool)

	for _, lookupCode := range Icons() {
		values[lookupCode.Value] = true
	}

	for _, legacy := range legacyIconValues {
		require.True(t, values[legacy], "value %q is stored in Domain records and must not be removed", legacy)
	}
}

func TestIcons_EveryIconExistsInTheVendoredSet(t *testing.T) {

	available := availableIcons(t)

	for _, lookupCode := range Icons() {

		// This is the name the select-icons widget writes as "bi-<name>"
		name := lookupCode.Icon

		if name == "" {
			name = lookupCode.Value
		}

		_, ok := available[name]
		require.True(t, ok, "icon %q (value %q) is not in Bootstrap Icons 1.13.1", name, lookupCode.Value)
	}
}

func TestIcons_PickerAndPageDrawTheSameIcon(t *testing.T) {

	provider := bootstrap.Provider{}

	for _, lookupCode := range Icons() {

		// What a template draws: {{icon .Value}} through the icon Provider
		matches := iconClassPattern.FindStringSubmatch(provider.Get(lookupCode.Value))
		require.Len(t, matches, 2, "provider returned unparsable markup for %q", lookupCode.Value)

		// What the select-icons widget draws
		widgetName := lookupCode.Icon

		if widgetName == "" {
			widgetName = lookupCode.Value
		}

		require.Equal(t, matches[1], widgetName,
			"value %q draws %q on the page but %q in the picker; set Icon to the resolved name",
			lookupCode.Value, matches[1], widgetName)
	}
}

func TestIcons_EveryCodeIsGroupedAndLabeled(t *testing.T) {

	for _, lookupCode := range Icons() {
		require.NotEmpty(t, lookupCode.Label, "value %q has no Label", lookupCode.Value)
		require.NotEmpty(t, lookupCode.Group, "value %q has no Group", lookupCode.Value)
	}
}

func TestIcons_ValuesAreUniqueAndGroupsAreContiguous(t *testing.T) {

	values := make(map[string]bool)
	seenGroups := make(map[string]bool)
	lastGroup := ""

	for _, lookupCode := range Icons() {

		require.False(t, values[lookupCode.Value], "duplicate value %q", lookupCode.Value)
		values[lookupCode.Value] = true

		// groupie emits one header per RUN, so a split group would draw two headers
		if lookupCode.Group != lastGroup {
			require.False(t, seenGroups[lookupCode.Group], "group %q is split into two runs", lookupCode.Group)
			seenGroups[lookupCode.Group] = true
			lastGroup = lookupCode.Group
		}
	}
}

func TestIcons_CallersCannotCorruptTheSharedList(t *testing.T) {

	// Callers sort lookup codes in place, so a shared backing array would be
	// reordered for the life of the process the first time one did.
	first := Icons()
	require.NotEmpty(t, first)

	original := first[0]
	slices.Reverse(first)

	second := Icons()
	require.NotEmpty(t, second)
	require.Equal(t, original, second[0], "Icons() handed out its own backing array")
}

// knownBrandMarks are a spot-check, not the full exclusion list. If any of these
// reappear, the brand filter in the generator regressed.
var knownBrandMarks = []string{
	"apple", "android", "google", "github", "facebook", "twitter", "youtube",
	"amazon", "windows", "linkedin", "spotify", "slack", "paypal", "stripe",
}

func TestIcons_AreOutlinesOnly(t *testing.T) {

	for _, lookupCode := range Icons() {

		name := lookupCode.Icon

		if name == "" {
			name = lookupCode.Value
		}

		// RULE: "flag" is the one exception. It is a legacy Value that Domain
		// records hold, and the icon Provider maps it to flag-fill.
		if lookupCode.Value == "flag" {
			continue
		}

		require.NotContains(t, strings.Split(name, "-"), "fill",
			"value %q draws the filled icon %q", lookupCode.Value, name)
	}
}

func TestIcons_ContainNoBrandMarks(t *testing.T) {

	values := make(map[string]bool)

	for _, lookupCode := range Icons() {
		values[lookupCode.Value] = true
	}

	for _, brand := range knownBrandMarks {
		require.False(t, values[brand], "brand mark %q must not be offered as a page icon", brand)
	}
}

func TestIcons_GroupsAreASmallSemanticSet(t *testing.T) {

	groups := make([]string, 0)

	for _, lookupCode := range Icons() {
		if !slices.Contains(groups, lookupCode.Group) {
			groups = append(groups, lookupCode.Group)
		}
	}

	// A-Z buckets were replaced by semantic groups; 38 groups is the regression
	require.LessOrEqual(t, len(groups), 20, "groups: %v", groups)
	require.GreaterOrEqual(t, len(groups), 5)

	for _, group := range groups {
		require.NotRegexp(t, `^([A-Z]|0-9)$`, group, "single-letter group %q is the old A-Z scheme", group)
	}
}

// badgeSuffixes are the marks Bootstrap stamps onto the corner of an existing
// glyph. They name an action, and a page icon names a thing.
var badgeSuffixes = []string{
	"add", "arrow-down", "arrow-up", "at", "check", "dash", "down",
	"exclamation", "gear", "heart", "hearts", "lock", "lock2", "minus",
	"plus", "slash", "star", "up", "x",
}

// badgedIconsWeKeep read as their own concept rather than as an action performed
// on the root, so both the root and the badged form are offered.
var badgedIconsWeKeep = []string{
	"arrow-down-up", "balloon-heart", "bell-slash", "box2-heart", "chat-heart",
	"chat-left-heart", "chat-right-heart", "chat-square-heart", "code-slash",
	"envelope-open-heart", "envelope-paper-heart", "eye-slash", "list-check",
	"postage-heart", "postcard-heart", "search-heart", "shield-check",
	"shield-lock", "sim-slash",
}

// longestRootOf returns the longest prefix of name that is itself an offered icon.
func longestRootOf(name string, offered map[string]bool) string {

	parts := strings.Split(name, "-")

	for size := len(parts) - 1; size > 0; size-- {

		if root := strings.Join(parts[:size], "-"); offered[root] {
			return root
		}
	}

	return ""
}

func TestIcons_NeverOfferARootAndItsBadgedTwin(t *testing.T) {

	offered := make(map[string]bool)

	for _, lookupCode := range Icons() {
		offered[lookupCode.Value] = true
	}

	for _, lookupCode := range Icons() {

		root := longestRootOf(lookupCode.Value, offered)

		if root == "" {
			continue
		}

		if !slices.Contains(badgeSuffixes, lookupCode.Value[len(root)+1:]) {
			continue
		}

		require.Contains(t, badgedIconsWeKeep, lookupCode.Value,
			"%q is %q with a badge on it; the grid shows no text, so the two tiles are near-identical",
			lookupCode.Value, root)
	}
}

func TestIcons_NeverOfferTwoValuesThatDrawTheSameGlyph(t *testing.T) {

	// A legacy Value and its resolved name are the trap here: "home" draws
	// house, so offering "house" too puts the same tile in the grid twice.
	drawnBy := make(map[string]string)

	for _, lookupCode := range Icons() {

		glyph := lookupCode.Icon

		if glyph == "" {
			glyph = lookupCode.Value
		}

		first, ok := drawnBy[glyph]
		require.False(t, ok, "%q and %q both draw %q", first, lookupCode.Value, glyph)

		drawnBy[glyph] = lookupCode.Value
	}
}

// documentTypeSuffixes are the formats Bootstrap hangs off "file-" and
// "file-earmark-". The matching "filetype-*" family names them outright.
var documentTypeSuffixes = []string{
	"bar-graph", "binary", "break", "code", "diff", "easel", "excel", "font",
	"image", "medical", "music", "pdf", "person", "play", "post", "ppt",
	"richtext", "ruled", "slides", "spreadsheet", "text", "word", "zip",
}

func TestIcons_OfferNoDocumentTypes(t *testing.T) {

	for _, lookupCode := range Icons() {

		require.False(t, strings.HasPrefix(lookupCode.Value, "filetype-"),
			"%q names a file format, not a page", lookupCode.Value)

		for _, root := range []string{"file-earmark-", "file-"} {

			if !strings.HasPrefix(lookupCode.Value, root) {
				continue
			}

			require.NotContains(t, documentTypeSuffixes, lookupCode.Value[len(root):],
				"%q names a file format, not a page", lookupCode.Value)

			break
		}
	}
}

// groupOrder is the order the picker draws, most likely to hold a page icon
// first. Changing it is a decision, not a refactor; see AGENTS.md.
var groupOrder = []string{
	"Files & Folders", "Media & Design", "People & Communication",
	"Text & Editing", "Time & Calendar", "Commerce & Money", "Places & Travel",
	"Symbols & Status", "Technology", "Objects & Tools", "Data & Charts",
	"Weather & Nature", "Security", "Health & Body", "Navigation & Layout",
}

func TestIcons_AreOrderedByHowLikelyAGroupIsToHoldAPageIcon(t *testing.T) {

	seen := make([]string, 0, len(groupOrder))

	for _, lookupCode := range Icons() {

		if (len(seen) == 0) || (seen[len(seen)-1] != lookupCode.Group) {
			seen = append(seen, lookupCode.Group)
		}
	}

	require.Equal(t, groupOrder, seen)
}
