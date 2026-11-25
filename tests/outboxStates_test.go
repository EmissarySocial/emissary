package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// outboxStates pins, for each group, every State that sends its Streams to followers, as "templateId:stateId".
// Emissary's own Templates load with every group.
var outboxStates = map[string][]string{
	"emissary": {
		"folder:published",
		"outbox-message:default",
		"outbox-message:published",
		"outbox-reply:default",
		"outbox-reply:published",
	},
	"atlas": {
		"atlas-post:published",
	},
	"bandwagon": {
		"bandwagon-album:published",
		"bandwagon-event:published",
		"bandwagon-news:published",
		"bandwagon-song:published",
	},
	"qwertylicious": {
		"qwerty-article:published",
		"qwerty-episode:published",
		"qwerty-event:published",
		"qwerty-photo:published",
		"qwerty-placemark:published",
		"qwerty-timeline-entry:published",
	},
}

// TestOutboxStates_ArePinned confirms that each group flags exactly the States listed in outboxStates,
// so that a Template cannot start or stop federating without a change here
func TestOutboxStates_ArePinned(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)

			expected := slices.Clone(outboxStates["emissary"])
			if group.Name != "emissary" {
				expected = append(expected, outboxStates[group.Name]...)
			}
			slices.Sort(expected)

			actual := make([]string, 0)

			for _, templateID := range loaded.Templates.Names() {
				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				for stateID, state := range template.States {
					if state.IsOutbox {
						actual = append(actual, templateID+":"+stateID)
					}
				}
			}

			slices.Sort(actual)
			require.Equal(t, expected, actual)
		})
	}
}

// TestOutboxStates_PublishStepsCarryNoOutboxAttribute confirms that no "save-and-publish" or "unpublish"
// step still sets `outbox`, which the State's flag has replaced and the steps no longer read
func TestOutboxStates_PublishStepsCarryNoOutboxAttribute(t *testing.T) {

	stepPattern := regexp.MustCompile(`do:\s*"?(save-and-publish|unpublish)"?[^}\n]*outbox`)

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			found := 0

			for _, location := range group.Locations {

				files, err := filepath.Glob(filepath.Join(location, "*", "*.hjson"))
				require.NoError(t, err)

				for _, file := range files {
					found++
					content, err := os.ReadFile(file) // #nosec G304 -- a template file found by the glob above
					require.NoError(t, err)
					require.False(t, stepPattern.Match(content), "%s still sets `outbox` on a publish step", file)
				}
			}

			// A silent zero would pass without checking anything
			require.NotZero(t, found, "no template files found for %s", group.Name)
		})
	}
}

// TestOutboxStates_ArticlesStayOut confirms that pages which are on the site and not sent carry no flag
func TestOutboxStates_ArticlesStayOut(t *testing.T) {

	loaded := loadGroup(t, templateGroups[0])

	for _, templateID := range []string{"article-base", "article-html", "article-markdown", "article-remote", "contact-form", "redirect"} {
		t.Run(templateID, func(t *testing.T) {
			template, err := loaded.Templates.Load(templateID)
			require.NoError(t, err)
			require.NotEmpty(t, template.States)

			for stateID, state := range template.States {
				require.False(t, state.IsOutbox, "state %q", stateID)
			}
		})
	}
}

// TestOutboxStates_SongWaitsForItsAlbum confirms that a song has a State outside the outbox, which only
// its artist can view, for songs whose album is not published
func TestOutboxStates_SongWaitsForItsAlbum(t *testing.T) {

	loaded := loadGroup(t, templateGroup{Name: "bandwagon", Locations: []string{emissaryTemplates, "../../bandwagon"}})

	song, err := loaded.Templates.Load("bandwagon-song")
	require.NoError(t, err)

	require.False(t, song.States["default"].IsOutbox)
	require.True(t, song.States["published"].IsOutbox)

	for _, actionID := range []string{"view", "prev", "next", "audio"} {
		action, exists := song.Actions[actionID]
		require.True(t, exists, actionID)
		require.NotContains(t, action.AccessList["default"], "anonymous", actionID)
		require.Contains(t, action.AccessList["published"], "anonymous", actionID)
	}
}
