package tests

import (
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
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

// TestOutboxStates_PublishStepsEnterFlaggedStates confirms that every "save-and-publish" step that turns the
// outbox on enters a State that carries the flag
func TestOutboxStates_PublishStepsEnterFlaggedStates(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			found := 0

			for _, templateID := range loaded.Templates.Names() {
				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				// RULE: Only a Stream is sent to followers
				if _, isStream := template.NewObject().(*model.Stream); !isStream {
					continue
				}

				for actionID, action := range template.Actions {
					for _, publish := range outboxPublishSteps(action.Steps) {
						found++
						state, exists := template.States[publish.StateID]
						require.True(t, exists, "%s/%s publishes into undefined state %q", templateID, actionID, publish.StateID)
						require.True(t, state.IsOutbox, "%s/%s publishes to the outbox in state %q, which has no outbox flag", templateID, actionID, publish.StateID)
					}
				}
			}

			// A silent zero would pass without checking anything
			require.NotZero(t, found, "no outbox publish steps found for %s", group.Name)
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

// outboxPublishSteps returns every "save-and-publish" step in the list, at any depth, that turns the outbox on
func outboxPublishSteps(steps []step.Step) []step.SaveAndPublish {

	result := make([]step.SaveAndPublish, 0)

	for _, current := range steps {

		if publish, isPublish := current.(step.SaveAndPublish); isPublish && publish.Outbox {
			result = append(result, publish)
		}

		for _, subSteps := range nestedSteps(current) {
			result = append(result, outboxPublishSteps(subSteps)...)
		}
	}

	return result
}
