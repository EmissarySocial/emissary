package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// iconSink keeps benchmark results alive so the compiler cannot discard the lookup
var iconSink string

// TestIcons_Get_Alias confirms that Emissary names resolve to their Bootstrap icons
func TestIcons_Get_Alias(t *testing.T) {

	icons := NewIcons()

	assert.Equal(t, `<i class="bi bi-plus-lg"></i>`, icons.Get("add"))
	assert.Equal(t, `<i class="bi bi-at"></i>`, icons.Get("at-fill"))
	assert.Equal(t, `<i class="bi bi-list-ul"></i>`, icons.Get("layout-social"))
	assert.Equal(t, `<i class="bi bi-stripe"></i>`, icons.Get("stripe"))
	assert.Equal(t, `<i class="bi bi-stripe"></i>`, icons.Get("STRIPE"))
}

// TestIcons_Get_Fallback confirms that unknown names pass through as Bootstrap icon names
func TestIcons_Get_Fallback(t *testing.T) {

	icons := NewIcons()

	assert.Equal(t, `<i class="bi bi-house-door"></i>`, icons.Get("house-door"))
	assert.Equal(t, `<i class="bi bi-"></i>`, icons.Get(""))

	// Lookups are case-sensitive, so this is not the "add" alias
	assert.Equal(t, `<i class="bi bi-ADD"></i>`, icons.Get("ADD"))
}

// TestIcons_Get_Composite confirms that the SVG and license icons return their full markup
func TestIcons_Get_Composite(t *testing.T) {

	icons := NewIcons()

	assert.Equal(t, "©", icons.Get("copyright"))
	assert.True(t, strings.HasPrefix(icons.Get("fediverse"), "<svg "))
	assert.Equal(t, icons.Get("fediverse"), icons.Get("fediverse-fill"))
	assert.Equal(t, icons.Get("activitypub"), icons.Get("activitypub-fill"))
	assert.Contains(t, icons.Get("bluesky"), `class="bi bi-bluesky"`)
	assert.Contains(t, icons.Get("webhooks"), "<svg ")
	assert.Contains(t, icons.Get("cc0"), "<svg ")
	assert.Contains(t, icons.Get("public-domain"), "<svg ")

	// Each license badge is a space-separated sequence of SVGs
	assert.Equal(t, 2, strings.Count(icons.Get("cc-by"), "<svg "))
	assert.Equal(t, 3, strings.Count(icons.Get("cc-by-sa"), "<svg "))
	assert.Equal(t, 3, strings.Count(icons.Get("cc-by-nc"), "<svg "))
	assert.Equal(t, 4, strings.Count(icons.Get("cc-by-nc-sa"), "<svg "))
	assert.Equal(t, 3, strings.Count(icons.Get("cc-by-nd"), "<svg "))
	assert.Equal(t, 4, strings.Count(icons.Get("cc-by-nc-nd"), "<svg "))
}

// TestIcons_Get_KnownNamesDoNotAllocate pins that a known name returns prebuilt markup
func TestIcons_Get_KnownNamesDoNotAllocate(t *testing.T) {

	icons := NewIcons()

	for _, name := range []string{"add", "stripe", "fediverse", "cc-by-nc-sa"} {
		allocs := testing.AllocsPerRun(100, func() { iconSink = icons.Get(name) })
		assert.Zero(t, allocs, "Get(%q) allocates", name)
	}
}

// TestIcons_NoNameInBothTables guards against an SVG silently replacing an alias of the same name
func TestIcons_NoNameInBothTables(t *testing.T) {

	for name := range iconSVGs {
		_, exists := iconAliases[name]
		assert.False(t, exists, "%q is defined in both iconAliases and iconSVGs", name)
	}

	require.Len(t, iconMarkup, len(iconAliases)+len(iconSVGs))
}

// TestIcons_Write confirms that Write emits the same markup as Get
func TestIcons_Write(t *testing.T) {

	icons := NewIcons()

	var buffer strings.Builder
	icons.Write("add", &buffer)
	icons.Write("house-door", &buffer)

	require.Equal(t, `<i class="bi bi-plus-lg"></i><i class="bi bi-house-door"></i>`, buffer.String())
}

// BenchmarkIcons_Get_Alias measures a lookup of an aliased Bootstrap icon
func BenchmarkIcons_Get_Alias(b *testing.B) {
	icons := NewIcons()
	for b.Loop() {
		iconSink = icons.Get("volume-mute-fill")
	}
}

// BenchmarkIcons_Get_Composite measures a lookup of a multi-badge license icon
func BenchmarkIcons_Get_Composite(b *testing.B) {
	icons := NewIcons()
	for b.Loop() {
		iconSink = icons.Get("cc-by-nc-sa")
	}
}

// BenchmarkIcons_Get_Fallback measures a lookup of a name with no alias
func BenchmarkIcons_Get_Fallback(b *testing.B) {
	icons := NewIcons()
	for b.Loop() {
		iconSink = icons.Get("house-door")
	}
}
