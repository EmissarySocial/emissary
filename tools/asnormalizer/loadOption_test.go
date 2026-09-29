package asnormalizer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewLoadConfig_MergesWithoutRepeats confirms that every WithHistory among the options is merged
// into one history, in order, with repeats and fragments removed.
func TestNewLoadConfig_MergesWithoutRepeats(t *testing.T) {

	config := newLoadConfig(
		WithHistory("https://example.com/a", "https://example.com/b"),
		WithHistory("https://example.com/b", "https://example.com/c#main-key"),
		WithHistory("https://example.com/c"),
	)

	require.Equal(t, []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}, config.history)
}

// TestNewLoadConfig_IgnoresOtherOptions confirms that options of other types, including nil, leave the
// history empty.
func TestNewLoadConfig_IgnoresOtherOptions(t *testing.T) {

	config := newLoadConfig("https://example.com/a", 42, nil, []string{"https://example.com/b"})

	require.Empty(t, config.history)
}

// TestNewLoadConfig_NoOptions confirms that a Load with no options starts with an empty history.
func TestNewLoadConfig_NoOptions(t *testing.T) {
	require.Empty(t, newLoadConfig().history)
}

// TestNewLoadConfig_SkipsEmptyURLs confirms that an empty URL, or one that is only a fragment, is
// never recorded.
func TestNewLoadConfig_SkipsEmptyURLs(t *testing.T) {

	config := newLoadConfig(WithHistory("", "#main-key"))

	require.Empty(t, config.history)
}

// TestNewLoadConfig_SiblingsDoNotShare confirms that two Loads given the same WithHistory each build
// their own history.
func TestNewLoadConfig_SiblingsDoNotShare(t *testing.T) {

	shared := WithHistory("https://example.com/a")

	first := newLoadConfig(shared, WithHistory("https://example.com/b"))
	second := newLoadConfig(shared, WithHistory("https://example.com/c"))

	require.Equal(t, []string{"https://example.com/a", "https://example.com/b"}, first.history)
	require.Equal(t, []string{"https://example.com/a", "https://example.com/c"}, second.history)
}

// TestWithHistory_KeepsPrivateCopy confirms that changing the caller's slice after the option is
// built does not change the history it adds.
func TestWithHistory_KeepsPrivateCopy(t *testing.T) {

	urls := []string{"https://example.com/a"}
	option := WithHistory(urls...)

	urls[0] = "https://example.com/changed"

	require.Equal(t, []string{"https://example.com/a"}, newLoadConfig(option).history)
}

// TestLoadConfig_Remember confirms that remember extends the history it is called on, with the same
// rules as WithHistory.
func TestLoadConfig_Remember(t *testing.T) {

	config := newLoadConfig(WithHistory("https://example.com/a"))
	config.remember("https://example.com/b#fragment", "https://example.com/a", "")

	require.Equal(t, []string{"https://example.com/a", "https://example.com/b"}, config.history)
}

// TestLoadConfig_Contains confirms that contains matches a URL with or without a fragment.
func TestLoadConfig_Contains(t *testing.T) {

	config := newLoadConfig(WithHistory("https://example.com/a#main-key"))

	require.True(t, config.contains("https://example.com/a"))
	require.True(t, config.contains("https://example.com/a#other"))
	require.False(t, config.contains("https://example.com/b"))
	require.False(t, config.contains(""))
}

// TestLoadConfig_IsTooDeep confirms that the history is too deep once it holds maxDepth URLs, and not
// one URL before.
func TestLoadConfig_IsTooDeep(t *testing.T) {

	config := loadConfig{}

	for index := range maxDepth {
		require.False(t, config.isTooDeep(), "history of %d", index)
		config.remember("https://example.com/" + string(rune('a'+index)))
	}

	require.True(t, config.isTooDeep())
}

// TestWithoutFragment confirms that only the fragment is removed, and only from the first "#".
func TestWithoutFragment(t *testing.T) {

	require.Equal(t, "https://example.com/a", withoutFragment("https://example.com/a"))
	require.Equal(t, "https://example.com/a", withoutFragment("https://example.com/a#main-key"))
	require.Equal(t, "https://example.com/a", withoutFragment("https://example.com/a#one#two"))
	require.Equal(t, "https://example.com/a?b=c", withoutFragment("https://example.com/a?b=c#d"))
	require.Empty(t, withoutFragment("#main-key"))
	require.Empty(t, withoutFragment(""))
}

// FuzzLoadConfig_Remember confirms that, for any URLs, the history holds no fragments, no empty
// values, and no repeats.
func FuzzLoadConfig_Remember(f *testing.F) {

	f.Add("https://example.com/a", "https://example.com/a#main-key")
	f.Add("", "#")
	f.Add("https://example.com/a#b#c", "https://example.com/a")
	f.Add("\xff\xfe", "not a url")

	f.Fuzz(func(t *testing.T, first string, second string) {

		config := newLoadConfig(WithHistory(first, second), WithHistory(second, first))

		seen := make(map[string]bool)

		for _, url := range config.history {
			require.NotEmpty(t, url)
			require.False(t, strings.Contains(url, "#"), "fragment kept in %q", url)
			require.False(t, seen[url], "repeated %q", url)
			seen[url] = true
		}

		require.LessOrEqual(t, len(config.history), 2)
	})
}
