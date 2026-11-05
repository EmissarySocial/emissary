package tests

import (
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

// TestUploadMultiple_WidgetAndStepAgree verifies that, in every template that manages files
// with an edit-attachments step, every html-remote element's url names an action on that
// template holding such a step, and that a create form's create action includes the action its
// url names, which is what lets the element reach the step before the Save and what adopts the
// staged files at the Save
func TestUploadMultiple_WidgetAndStepAgree(t *testing.T) {

	// html-remote is a plain loader, and nothing but this test connects the URL it names to the
	// step behind it; see "An html-remote element and its edit-attachments step are linked by
	// nothing but a test" in _embed/templates/AGENTS.md.

	// Groups known to use the step, so that a count of zero there means the test is watching nothing
	adopters := map[string]bool{"qwertylicious": true}

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			checked := 0

			for _, templateID := range loaded.templateIDs() {

				template, err := loaded.Templates.Load(templateID)
				require.NoError(t, err)

				// RULE: Only a template that manages files is held to this; an html-remote elsewhere may load anything
				if !managesFiles(&template) {
					continue
				}

				for _, actionID := range sortedKeys(template.Actions) {
					for _, element := range elementsOfType(template.Actions[actionID].Steps, "html-remote") {
						checkRemoteLoader(t, &template, actionID, element)
						checked++
					}
				}
			}

			if adopters[group.Name] {
				require.Positive(t, checked, "no template in %s loads a file manager through html-remote -- this test is watching nothing", group.Name)
			}
		})
	}
}

// checkRemoteLoader verifies that one element's url names an action on its own template that
// holds an edit-attachments step, and that a create form's create action includes that action
func checkRemoteLoader(t *testing.T, template *model.Template, actionID string, element form.Element) {

	t.Helper()

	where := template.TemplateID + "/" + actionID

	// RULE: Every element needs a url, because it has nowhere else to load from
	remoteURL := element.Options.GetString("url", nil)
	require.NotEmpty(t, remoteURL, "%s: html-remote needs a url option", where)

	// The action is the url's ?action= on the parent's add, and its last path segment otherwise
	targetID := actionOf(t, where, remoteURL)

	target, exists := template.Action(targetID)
	require.True(t, exists, "%s: url %q names action %q, which %s does not define", where, remoteURL, targetID, template.TemplateID)

	_, found := findEditAttachments(template, target.Steps, map[string]bool{})
	require.True(t, found, "%s: action %q holds no edit-attachments step", where, targetID)

	// RULE: A create form reaches the step before the Save only through an action its create
	// action includes, and the include is what adopts the staged files at the Save
	if actionID == "create" {
		require.True(t, strings.HasSuffix(pathOf(remoteURL), "/add"), "%s: a create form's url must be the parent's add with ?action=, because the object has no route of its own yet; got %q", where, remoteURL)
		require.True(t, includesAction(template.Actions[actionID].Steps, targetID), "%s: the create action must include action %q, directly or inside a nested pipeline, or the element's requests are refused before the Save and the staged files are dropped at it", where, targetID)
	}
}

// actionOf returns the action a loader's url names: the ?action= of a parent's add, or the url's last path segment
func actionOf(t *testing.T, where string, remoteURL string) string {

	t.Helper()

	urlPath, rawQuery, _ := strings.Cut(remoteURL, "?")
	query, err := url.ParseQuery(rawQuery)
	require.NoError(t, err, "%s: url %q has a query that does not parse", where, remoteURL)

	if strings.HasSuffix(urlPath, "/add") {
		require.NotEmpty(t, query.Get("action"), "%s: url %q is the parent's add and must name the action in ?action=", where, remoteURL)
		return query.Get("action")
	}

	return path.Base(urlPath)
}

// managesFiles reports whether any action of the template holds an edit-attachments step
func managesFiles(template *model.Template) bool {

	for _, actionID := range sortedKeys(template.Actions) {
		if _, found := findEditAttachments(template, template.Actions[actionID].Steps, map[string]bool{}); found {
			return true
		}
	}

	return false
}

// includesAction reports whether a pipeline, or one nested in its steps, runs the named action through include
func includesAction(steps []step.Step, actionID string) bool {

	for _, current := range steps {

		if include, ok := current.(step.Include); ok && include.Action == actionID {
			return true
		}

		for _, subSteps := range nestedSteps(current) {
			if includesAction(subSteps, actionID) {
				return true
			}
		}
	}

	return false
}

// elementsOfType returns every element of the type in the forms a pipeline renders, searching
// nested pipelines too
func elementsOfType(steps []step.Step, elementType string) []form.Element {

	result := make([]form.Element, 0)

	for _, current := range steps {

		if formGetter, ok := current.(step.FormGetter); ok {
			result = append(result, elementsBeneath(formGetter.GetForm(), elementType)...)
		}

		for _, subSteps := range nestedSteps(current) {
			result = append(result, elementsOfType(subSteps, elementType)...)
		}
	}

	return result
}

// elementsBeneath returns the element and every element beneath it whose Type matches
func elementsBeneath(element form.Element, elementType string) []form.Element {

	result := make([]form.Element, 0)

	if element.Type == elementType {
		result = append(result, element)
	}

	for _, child := range element.Children {
		result = append(result, elementsBeneath(child, elementType)...)
	}

	return result
}

// findEditAttachments returns the first edit-attachments step in a pipeline, searching nested
// pipelines and the actions that include steps run, or FALSE
func findEditAttachments(template *model.Template, steps []step.Step, visited map[string]bool) (step.EditAttachments, bool) {

	for _, current := range steps {

		switch typed := current.(type) {

		case step.EditAttachments:
			return typed, true

		// An include runs another action's pipeline in place, once per action, so a loop ends here
		case step.Include:

			if visited[typed.Action] {
				continue
			}

			visited[typed.Action] = true

			if action, exists := template.Action(typed.Action); exists {
				if result, found := findEditAttachments(template, action.Steps, visited); found {
					return result, true
				}
			}
		}

		for _, subSteps := range nestedSteps(current) {
			if result, found := findEditAttachments(template, subSteps, visited); found {
				return result, true
			}
		}
	}

	return step.EditAttachments{}, false
}

// pathOf returns a url without its query string
func pathOf(remoteURL string) string {
	result, _, _ := strings.Cut(remoteURL, "?")
	return result
}
