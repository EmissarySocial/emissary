package tests

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartupTasks_CanBeCompleted verifies that every task a Theme lists links to this website, and
// names a connection provider or a value that some action records
func TestStartupTasks_CanBeCompleted(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			recorded := recordedStartupTasks(t, loaded)
			require.NotEmpty(t, recorded, "no action records a startup task")

			startupTemplate, err := loaded.Templates.Load("admin-startup")
			require.NoError(t, err)

			providers := map[string]bool{}

			for _, provider := range dataset.Providers() {
				providers[provider.Value] = true
			}

			for _, theme := range loaded.themes() {

				// RULE: A Domain records at most 16 tasks, of at most 32 characters each (model.DomainSchema)
				assert.LessOrEqual(t, len(theme.StartupTasks), 16, "%s lists more tasks than a Domain can record", theme.ThemeID)

				for _, task := range theme.StartupTasks {
					where := theme.ThemeID + " task " + task.Label

					assert.NotEmpty(t, task.Label, "%s: a task has no label", theme.ThemeID)
					assert.True(t, strings.HasPrefix(task.Href, "/"), "%s must link to a page on this website", where)
					assert.LessOrEqual(t, len(task.Value), 32, "%s has a value too long to record", where)

					// RULE: A link into the checklist's own modals names an action that exists
					if action, isStartup := strings.CutPrefix(task.Href, "/startup/"); isStartup {
						assert.Contains(t, startupTemplate.Actions, action, "%s links to a startup action that does not exist", where)
					}
					assert.True(t, recorded[task.Value] || providers[task.Value], "%s (%s) can never be checked off", where, task.Value)
				}
			}
		})
	}
}

// TestStartupStreams_UseLoadedTemplates verifies that every Stream a Theme seeds names a token and a Template that loads
func TestStartupStreams_UseLoadedTemplates(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)

			for _, theme := range loaded.themes() {

				tokens := map[string]bool{}
				templates := map[string]model.Template{}

				for _, stream := range theme.StartupStreams {

					token := stream.GetString("token")
					where := theme.ThemeID + " startup stream " + token

					assert.NotEmpty(t, token, "%s: a startup stream has no token", theme.ThemeID)
					assert.False(t, tokens[token], "%s is listed twice", where)
					tokens[token] = true

					template, err := loaded.Templates.Load(stream.GetString("templateId"))
					assert.NoError(t, err, "%s names a template that does not load", where)

					// RULE: A sub-page is listed after its parent, whose template must be able to contain it
					if parentToken := stream.GetString("parentToken"); parentToken != "" {
						parentTemplate, exists := templates[parentToken]
						assert.True(t, exists, "%s names the parent %s, which is not listed before it", where, parentToken)
						assert.True(t, template.CanBeContainedBy(parentTemplate.TemplateRole), "%s cannot be a sub-page of %s", where, parentToken)
					}

					templates[token] = template
				}
			}
		})
	}
}

// TestStartupDefaultPages_PointAtSeededPages verifies that each landing page a Theme sets is stored whole,
// and is a built-in page or one of the Streams that the same Theme seeds
func TestStartupDefaultPages_PointAtSeededPages(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)

			for _, theme := range loaded.themes() {

				pages := theme.StartupDefaultPages

				if pages.IsEmpty() {
					continue
				}

				// RULE: Apply stores only a path on this website, so a refused page would be silently skipped
				writableDomain := model.NewWritableDomain()
				pages.Apply(&writableDomain)
				assert.Equal(t, pages.Anonymous, writableDomain.DefaultAnonymous, theme.ThemeID)
				assert.Equal(t, pages.Authenticated, writableDomain.DefaultAuthenticated, theme.ThemeID)
				assert.Equal(t, pages.Owner, writableDomain.DefaultOwner, theme.ThemeID)

				seeded := map[string]bool{}

				for _, stream := range theme.StartupStreams {
					seeded["/"+stream.GetString("token")] = true
				}

				for _, page := range []string{pages.Anonymous, pages.Authenticated, pages.Owner} {

					if (page == "") || strings.HasPrefix(page, "/@me") || strings.HasPrefix(page, "/admin") {
						continue
					}

					assert.True(t, seeded[page], "%s sets the landing page %s, which its startup streams do not create", theme.ThemeID, page)
				}

				// RULE: A visitor who is not signed in can only land on a page that the Theme seeds
				assert.True(t, (pages.Anonymous == "") || seeded[pages.Anonymous], "%s sends anonymous visitors to %s, which they cannot open", theme.ThemeID, pages.Anonymous)
			}
		})
	}
}

// recordedStartupTasks returns the value of every startup-save-task step in every loaded Template
func recordedStartupTasks(t *testing.T, loaded *loadedGroup) map[string]bool {

	t.Helper()

	result := map[string]bool{}

	for _, templateID := range loaded.templateIDs() {

		template, err := loaded.Templates.Load(templateID)
		require.NoError(t, err)

		for _, action := range template.Actions {
			collectStartupTasks(action.Steps, result)
		}
	}

	return result
}

// collectStartupTasks adds the value of every startup-save-task step in a pipeline, and in the pipelines nested in it
func collectStartupTasks(steps []step.Step, result map[string]bool) {

	for _, current := range steps {

		if saveTask, ok := current.(step.StartupSaveTask); ok {
			result[saveTask.Value] = true
		}

		for _, subSteps := range nestedSteps(current) {
			collectStartupTasks(subSteps, result)
		}
	}
}
