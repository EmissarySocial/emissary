package service

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Remote Article Template
 *
 * An article whose body is a Markdown file hosted
 * somewhere else.  It EXTENDS article-base, and
 * inheritance is additive -- a child may override
 * an action but can never remove one -- so most of
 * what these tests pin is what survived, or was
 * neutralized, rather than what was written.
 ******************************************/

// articleRemoteTemplate returns the RESOLVED definition, after inheritance from article-base.
// Reading the hjson alone would prove nothing here: every property that matters is one the
// parent also declares, so only the merged result says which one won.
func articleRemoteTemplate(t *testing.T) model.Template {

	t.Helper()

	result, err := loadEmbeddedTemplates(t).Load("article-remote")
	require.NoError(t, err)

	return result
}

// TestArticleRemoteTemplate_ContentHTMLIsUnsafeAny pins the property that decides whether syntax
// highlighting and embeds survive a sync.  article-base declares this same property as `html`, and
// rosetta's `html` format re-sanitizes on every save with a plain UGCPolicy that allows neither --
// while the Markdown renderer already sanitized it with Emissary's own policy, which calls
// AllowStyling() and permits iframe.  Losing the override deletes both, and reports nothing.
func TestArticleRemoteTemplate_ContentHTMLIsUnsafeAny(t *testing.T) {

	template := articleRemoteTemplate(t)

	for _, path := range []string{"content.html", "content.raw"} {

		element, exists := template.Schema.GetElement(path)
		require.True(t, exists, "%s must be declared", path)

		stringElement, isString := element.(schema.String)
		require.True(t, isString, "%s must be a string", path)
		require.Equal(t, "unsafe-any", stringElement.Format, "%s lost its override of article-base", path)
	}
}

// TestArticleRemoteTemplate_TagsAreEnabled pins a pair that is useless by halves.  Template.Inherit
// deliberately does not pass TagPaths down, so extending article-base brings none -- and Stream.Save
// skips CalculateTags entirely when the list is empty, which turns the Hashtags field into a value
// that is stored and never read.  Declaring the paths without offering the field is the same dead
// configuration from the other side, so both are checked here.
func TestArticleRemoteTemplate_TagsAreEnabled(t *testing.T) {

	template := articleRemoteTemplate(t)

	require.Equal(t, []string{"data.tags"}, template.TagPaths, "tags are extracted from the same path as every other article")
	require.NotEmpty(t, template.TagURL, "linkification is silently skipped without a tag URL")

	_, exists := template.Schema.GetElement("data.tags")
	require.True(t, exists, "data.tags did not arrive from article-base")

	require.True(t, articleRemoteFormOffers(t, template, "data.tags"), "the Article Info form offers no way to type a hashtag")
}

// articleRemoteAllSteps flattens a pipeline, descending into the container steps this Template
// uses.  article-base wraps almost everything in `with-draft`, so a search of the top level alone
// reports that an action does nothing at all.
func articleRemoteAllSteps(steps []modelStep.Step) []modelStep.Step {

	result := make([]modelStep.Step, 0, len(steps))

	for _, step := range steps {

		result = append(result, step)

		switch typed := step.(type) {

		case modelStep.WithDraft:
			result = append(result, articleRemoteAllSteps(typed.SubSteps)...)

		case modelStep.AsModal:
			result = append(result, articleRemoteAllSteps(typed.SubSteps)...)

		case modelStep.WithStreamSource:
			result = append(result, articleRemoteAllSteps(typed.SubSteps)...)
		}
	}

	return result
}

// articleRemoteFormOffers reports whether the `properties` action's form edits the given path.
func articleRemoteFormOffers(t *testing.T, template model.Template, path string) bool {

	t.Helper()

	action, exists := template.Actions["properties"]
	require.True(t, exists, "action `properties` is missing")

	var offeredBy func(element form.Element) bool

	offeredBy = func(element form.Element) bool {

		if element.Path == path {
			return true
		}

		return slices.ContainsFunc(element.Children, offeredBy)
	}

	for _, step := range articleRemoteAllSteps(action.Steps) {

		if edit, isEdit := step.(modelStep.EditModelObject); isEdit {

			if offeredBy(edit.Form) {
				return true
			}
		}
	}

	return false
}

// TestArticleRemoteTemplate_ShipsNoReachableContentEditor confirms that nothing offers to edit the
// body.  The remote file is the source of truth, so an editable page invites a change that the
// next synchronization discards without saying so.
//
// These two actions cannot be deleted -- inheritance only ever adds -- so the template overrides
// each one to name no roles, which empties its access list and hides it from the menu bar.  The
// `forward-to` covers the Domain Owner, for whom every permission check answers true regardless.
//
// `promote-draft` and `discard-draft` are deliberately NOT in this list.  An earlier revision
// neutralized them, which took away the only path to save-and-publish and left the article
// unreachable from every auto-generated navigation list.  Promote is now the single control, and
// it publishes everything EXCEPT the body -- see TestArticleRemoteTemplate_PromoteKeepsTheBody.
func TestArticleRemoteTemplate_ShipsNoReachableContentEditor(t *testing.T) {

	template := articleRemoteTemplate(t)

	for _, actionID := range []string{"editor", "upload-image"} {

		action, exists := template.Actions[actionID]
		require.True(t, exists, "action %q should still be inherited", actionID)
		require.Empty(t, action.Roles, "action %q must name no roles", actionID)

		// CalcAccessList has already run, so this is the list the permission service reads
		for stateID := range template.States {
			require.Empty(t, action.AccessList[stateID], "action %q is reachable in state %q", actionID, stateID)
		}
	}

	// ..and no action anywhere may still run the steps that write a body
	for actionID, action := range template.Actions {
		for _, step := range articleRemoteAllSteps(action.Steps) {
			require.NotContains(t, []string{"edit-content", "upload-attachments"}, step.Name(), "action %q edits the body", actionID)
		}
	}
}

// TestArticleRemoteTemplate_PromoteKeepsTheBody pins the one attribute that stops Promote from
// undoing a synchronization.  A draft is a snapshot taken when it was created -- and `edit` runs
// inside with-draft, so merely opening the settings screen makes one -- so promoting copies that
// stale body over whatever the last sync fetched, or over nothing at all if the draft predates the
// first sync.  Nothing repairs it: the StreamSource's ContentHash still matches what it last wrote,
// so the next sync stops before fetching and Sync Now does nothing.
//
// The second half guards the restatement.  Template.Inherit replaces an action wholesale, so adding
// `omit` here meant copying article-base's whole pipeline by hand -- and a hand copy goes stale
// silently when the parent changes.
func TestArticleRemoteTemplate_PromoteKeepsTheBody(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	names := func(templateID string) []string {

		template, err := templateService.Load(templateID)
		require.NoError(t, err)

		action, exists := template.Actions["promote-draft"]
		require.True(t, exists, "%s has no promote-draft action", templateID)

		result := make([]string, 0, len(action.Steps))

		for _, step := range action.Steps {

			if promote, isPromote := step.(modelStep.StreamPromoteDraft); isPromote && templateID == "article-remote" {
				require.Contains(t, promote.Omit, "content", "Promote would copy the draft's stale body over the live Stream")
			}

			result = append(result, step.Name())
		}

		return result
	}

	require.Equal(t, names("article-base"), names("article-remote"),
		"article-remote restates article-base's promote-draft by hand, and the two have drifted")
}

// TestArticleRemoteTemplate_LayoutActionsUseDrafts confirms that four inherited actions are left
// alone.  Each wraps its work in `with-draft`, and a draft reaches the live page only through
// promote-draft -- which this Template now runs in full.  An override here would be the old design,
// where promote-draft was neutralized and a layout or stylesheet change saved into a draft that
// nothing ever published: success reported, nothing visible, no error anywhere.
func TestArticleRemoteTemplate_LayoutActionsUseDrafts(t *testing.T) {

	template := articleRemoteTemplate(t)

	for _, actionID := range []string{"widgets", "widget", "style", "properties"} {

		action, exists := template.Actions[actionID]
		require.True(t, exists, "action %q is missing", actionID)

		usesDraft := slices.ContainsFunc(action.Steps, func(step modelStep.Step) bool {
			_, isDraft := step.(modelStep.WithDraft)
			return isDraft
		})

		require.True(t, usesDraft, "action %q must save into the draft that Promote publishes", actionID)
	}
}

// TestArticleRemoteTemplate_InheritsLayoutAndPublishing confirms the reason this Template extends
// article-base at all -- every name below arrives only by inheritance.
//
// A MISSPELLED parent is caught by the loader, which refuses to start.  A DROPPED `extends` is
// not: calculateInheritance returns early when the list is empty, so the Template would load
// clean and quietly lose its layout, widgets, and publish workflow.
func TestArticleRemoteTemplate_InheritsLayoutAndPublishing(t *testing.T) {

	template := articleRemoteTemplate(t)

	require.Contains(t, template.Extends, "article-base")

	for _, actionID := range []string{"view", "heading", "widgets", "widget", "style", "children", "publish", "unpublish", "sharing"} {
		_, exists := template.Actions[actionID]
		require.True(t, exists, "action %q did not arrive from article-base", actionID)
	}

	// The widget editor and the width control read these, and neither is declared here
	for _, path := range []string{"data.width", "data.stylesheet"} {
		_, exists := template.Schema.GetElement(path)
		require.True(t, exists, "%s did not arrive from article-base", path)
	}

	require.NotEmpty(t, template.WidgetLocations, "widgets have nowhere to go")
}

// TestArticleRemoteTemplate_SettingsAreReachable pins the four things D11 requires an operator be
// able to see and do.  Every failure mode this design accepts is silent, so a record nobody
// finished configuring has to read as unconfigured rather than as working.
func TestArticleRemoteTemplate_SettingsAreReachable(t *testing.T) {

	template := articleRemoteTemplate(t)

	for _, actionID := range []string{"edit", "edit-source", "sync-source"} {
		action, exists := template.Actions[actionID]
		require.True(t, exists, "action %q is missing", actionID)
		require.NotEmpty(t, action.Roles, "action %q is defined but reachable by nobody", actionID)
	}

	// `edit` is the settings screen: it renders THIS Template's page, never article-base's body
	// editor.  The with-draft wrapper it inherits is harmless, because the settings it shows come
	// from the StreamSource record, which lives outside the Stream entirely.
	for _, step := range articleRemoteAllSteps(template.Actions["edit"].Steps) {
		require.NotEqual(t, "edit-content", step.Name(), "edit renders article-base's body editor")
	}
}

// TestArticleRemoteTemplate_SyncNowIsJustASave pins the mechanism behind the Sync Now button.
// Saving the record IS the request to synchronize -- there is no step and no flag -- so this
// pipeline must stay empty of anything that looks like it does the work.
func TestArticleRemoteTemplate_SyncNowIsJustASave(t *testing.T) {

	action, exists := articleRemoteTemplate(t).Actions["sync-source"]
	require.True(t, exists)
	require.Len(t, action.Steps, 1)

	container, isContainer := action.Steps[0].(modelStep.WithStreamSource)
	require.True(t, isContainer, "Sync Now must run against the StreamSource record")

	names := make([]string, 0, len(container.SubSteps))

	for _, subStep := range container.SubSteps {
		names = append(names, subStep.Name())
	}

	// NOT "refresh-page": `save` publishes the SSE nudge that redraws this screen, and a second
	// trigger for the same write swapped <main> twice per press.
	require.Equal(t, []string{"save"}, names)
}

// TestArticleRemoteTemplate_ViewIsPublishedOnly confirms that an unpublished article is not
// public.  The sync never changes state (D4), so an article stays wherever its author left it.
func TestArticleRemoteTemplate_ViewIsPublishedOnly(t *testing.T) {

	action, exists := articleRemoteTemplate(t).Actions["view"]
	require.True(t, exists)

	require.NotContains(t, action.Roles, "viewer", "a viewer must earn access through the published state")
	require.Contains(t, action.StateRoles["published"], "viewer")
}

// TestArticleRemoteTemplate_IsOfferedInThePicker confirms the template reaches the "Add a Page"
// dialog under every container it claims.  That dialog reads ListByContainerLimited, which drops a
// template SILENTLY when containedBy or templateRole disagree with the parent -- there is no error
// and no log line, just a button that never appears.
func TestArticleRemoteTemplate_IsOfferedInThePicker(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	offers := func(listed sliceof.Object[form.LookupCode]) bool {
		return slices.ContainsFunc(listed, func(lookupCode form.LookupCode) bool {
			return lookupCode.Value == "article-remote"
		})
	}

	for _, containedBy := range []string{"top", "home", "folder", "article"} {
		require.True(t, offers(templateService.ListByContainer(containedBy)), "not offered inside %q", containedBy)
	}

	// The folder and article pickers name no template roles, so an unlimited list must match
	require.True(t, offers(templateService.ListByContainerLimited("folder", nil)), "not offered by an unlimited picker")

	// ..and a picker that limits to `article` roles must still find it
	require.True(t, offers(templateService.ListByContainerLimited("folder", []string{"article"})), "templateRole is not `article`")

	// A container it does NOT claim must not offer it
	require.False(t, offers(templateService.ListByContainer("outbox")), "offered inside a container it does not claim")
}

// TestArticleRemoteTemplate_SyncButtonSwapsNothing pins the attribute that keeps the Sync button
// from blanking the page.  htmx attributes INHERIT, and theme-default's <body> carries
// hx-target="main" with hx-swap="innerHTML transition:true" -- while the sync-source pipeline
// answers with an EMPTY body and a pair of HX-Trigger headers.  Without hx-swap="none" htmx
// swaps that empty body into <main>, so the whole page clears, with a view transition, until the
// refresh lands.  Nothing errors; it just looks like the screen jumped.
func TestArticleRemoteTemplate_SyncButtonSwapsNothing(t *testing.T) {

	page, err := os.ReadFile("../_embed/templates/stream-article-remote/edit.html")
	require.NoError(t, err)

	// Every element that POSTs from this page runs a pipeline that renders nothing
	for _, line := range strings.Split(string(page), "\n") {

		if !strings.Contains(line, "hx-post=") {
			continue
		}

		require.Contains(t, line, `hx-swap="none"`, "an hx-post here must not swap its empty response into <main>: %s", strings.TrimSpace(line))
		require.Contains(t, line, `hx-push-url="false"`, "..and must not put its action URL in the address bar: %s", strings.TrimSpace(line))
	}
}
