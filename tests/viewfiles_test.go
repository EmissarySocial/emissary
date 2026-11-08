package tests

import (
	"html/template"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knownMissingViewFiles lists the views, as templateID/actionID, that name a file their template
// lacks, with the bug each waits on. A fixed one must leave the list.
var knownMissingViewFiles = map[string]string{
	"admin-navigation/view":                    "BUG-346",
	"admin-startup/choose-theme":               "BUG-346",
	"admin-webhooks/list":                      "BUG-346",
	"user-settings/merchantAccounts":           "BUG-346",
	"user-settings/merchantAccounts-list":      "BUG-346",
	"atlas-search/view-suggestions":            "BUG-346",
	"bandwagon-search-albums/view-albums":      "BUG-346",
	"bandwagon-search-tracks/view-albums-list": "BUG-346",
}

// TestViewFiles_Exist verifies that every view-html and view-css in every action names a file
// its template holds, so that no action can fail to render for want of one
func TestViewFiles_Exist(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			missing := map[string]bool{}
			loadedIDs := map[string]bool{}

			// Check every action of every Template
			for _, templateID := range loaded.templateIDs() {

				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				loadedIDs[templateID] = true
				checkViewFiles(t, templateID, template.Actions, template.HTMLTemplate, missing)
			}

			// Check every action of every Registration
			for _, code := range loaded.Registrations.List() {

				registration, err := loaded.Registrations.Load(code.Value)
				require.NoError(t, err)

				loadedIDs[code.Value] = true
				checkViewFiles(t, code.Value, registration.Actions, registration.HTMLTemplate, missing)
			}

			// RULE: A view outside the known list must render, and one on it must still fail, so the list stays true
			for _, where := range sortedKeys(missing) {
				_, known := knownMissingViewFiles[where]
				assert.True(t, known, "%s: a view names a file its template does not have", where)
			}

			for _, where := range sortedKeys(knownMissingViewFiles) {

				if templateID, _, _ := strings.Cut(where, "/"); loadedIDs[templateID] {
					assert.True(t, missing[where], "%s renders now, so remove it from knownMissingViewFiles (%s)", where, knownMissingViewFiles[where])
				}
			}
		})
	}
}

// checkViewFiles records, as templateID/actionID, each action holding a view whose file the
// template set lacks, and fails on a view the loader left without a file
func checkViewFiles(t *testing.T, templateID string, actions map[string]model.Action, files *template.Template, missing map[string]bool) {

	t.Helper()

	for _, actionID := range sortedKeys(actions) {

		where := templateID + "/" + actionID

		for _, file := range viewFiles(actions[actionID].Steps) {

			// RULE: The loader names an action's own file on every view that names none
			if !assert.NotEmpty(t, file, "%s: a view has no file, so the loader did not name its default", where) {
				continue
			}

			if files.Lookup(file) == nil {
				missing[where] = true
			}
		}
	}
}

// viewFiles returns the file of every view-html and view-css in an action's own steps, searching
// nested pipelines but not a with-* step's, which renders another record's template, nor an include's
func viewFiles(steps []step.Step) []string {

	result := make([]string, 0)

	for _, current := range steps {

		switch typed := current.(type) {

		case step.ViewHTML:
			result = append(result, typed.File)

		case step.ViewCSS:
			result = append(result, typed.File)
		}

		if strings.HasPrefix(current.Name(), "with-") {
			continue
		}

		for _, subSteps := range nestedSteps(current) {
			result = append(result, viewFiles(subSteps)...)
		}
	}

	return result
}

// TestIncludes_LoaderActionsRunOnPostOnly verifies that, when a form loads an action through an
// html-remote element on the parent's add, the form's action includes that action on the POST
// alone, because on the GET the element's own request draws it and an include would draw it twice
func TestIncludes_LoaderActionsRunOnPostOnly(t *testing.T) {

	// Groups known to load an action this way, so that a count of zero there means the test is watching nothing
	adopters := map[string]bool{"qwertylicious": true}

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			checked := 0

			for _, templateID := range loaded.templateIDs() {

				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				for _, actionID := range sortedKeys(template.Actions) {

					steps := template.Actions[actionID].Steps
					where := templateID + "/" + actionID

					for _, element := range elementsOfType(steps, "html-remote") {

						remoteURL := element.Options.GetString("url", nil)

						// RULE: Only a loader on the parent's add stands in for the include on the GET
						if !strings.HasSuffix(pathOf(remoteURL), "/add") {
							continue
						}

						targetID := actionOf(t, where, remoteURL)
						includes := includesOf(steps, targetID)

						require.NotEmpty(t, includes, "%s: the action must include %q, or the parent's add refuses the element's request", where, targetID)

						for _, include := range includes {
							assert.Equal(t, "post", include.Method, "%s: the include of %q must run on the POST only, or the GET draws it a second time", where, targetID)
						}

						checked++
					}
				}
			}

			if adopters[group.Name] {
				require.Positive(t, checked, "no form in %s loads an action through the parent's add -- this test is watching nothing", group.Name)
			}
		})
	}
}

// includesOf returns every include of the named action in a pipeline, searching nested pipelines too
func includesOf(steps []step.Step, actionID string) []step.Include {

	result := make([]step.Include, 0)

	for _, current := range steps {

		if include, ok := current.(step.Include); ok && include.Action == actionID {
			result = append(result, include)
		}

		for _, subSteps := range nestedSteps(current) {
			result = append(result, includesOf(subSteps, actionID)...)
		}
	}

	return result
}
