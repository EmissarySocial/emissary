package service

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// adminStartupBuilder is the smallest dot that the startup checklist renders against
type adminStartupBuilder struct {
	themes       sliceof.Object[model.Theme]    // Every visible Theme. The first is the Domain's own
	startupTasks sliceof.String                 // Tasks recorded on the Domain
	connections  mapof.Object[model.Connection] // Connections, keyed by provider
}

// newAdminStartupBuilder returns a stub whose current Theme lists the provided tasks
func newAdminStartupBuilder(themeCount int, tasks ...form.LookupCode) adminStartupBuilder {

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	themes := sliceof.NewObject[model.Theme]()

	for _, themeID := range []string{"default", "minimal", "extra"}[:themeCount] {
		theme := model.NewTheme(themeID, funcMap)
		theme.Label = "Theme " + themeID
		themes = append(themes, theme)
	}

	themes[0].StartupTasks = tasks

	return adminStartupBuilder{
		themes:       themes,
		startupTasks: sliceof.NewString(),
		connections:  mapof.NewObject[model.Connection](),
	}
}

// StartupTasks returns the tasks recorded on the Domain
func (builder adminStartupBuilder) StartupTasks() sliceof.String {
	return builder.startupTasks
}

// Themes returns every visible Theme
func (builder adminStartupBuilder) Themes() sliceof.Object[model.Theme] {
	return builder.themes
}

// ThemeID returns the ID of the Domain's own Theme
func (builder adminStartupBuilder) ThemeID() string {
	return builder.themes[0].ThemeID
}

// Theme returns the Domain's own Theme for every ID, which is all the checklist asks for
func (builder adminStartupBuilder) Theme(_ string) model.Theme {
	return builder.themes[0]
}

// AllConnections returns the Domain's Connections, keyed by provider
func (builder adminStartupBuilder) AllConnections() mapof.Object[model.Connection] {
	return builder.connections
}

// renderAdminStartup executes one admin-startup HTML template against the stub builder
func renderAdminStartup(t *testing.T, file string, builder any) string {

	t.Helper()

	var buffer bytes.Buffer
	templateService := loadProfileTabsTemplates(t, nil)
	adminStartup := templateService.templatePrep["admin-startup"]
	require.NoError(t, adminStartup.HTMLTemplate.ExecuteTemplate(&buffer, file, builder))

	return buffer.String()
}

// TestAdminStartup_PickerNeedsASecondTheme confirms that the theme picker is offered only when there is a choice
func TestAdminStartup_PickerNeedsASecondTheme(t *testing.T) {

	html := renderAdminStartup(t, "page", newAdminStartupBuilder(1))
	require.NotContains(t, html, "/admin/domain/themes")

	html = renderAdminStartup(t, "page", newAdminStartupBuilder(2))
	require.Contains(t, html, `hx-get="/admin/domain/themes"`)
	require.Contains(t, html, "Theme default")
}

// TestAdminStartup_TaskIsDoneWhenRecorded confirms that a recorded task is checked and the others are not
func TestAdminStartup_TaskIsDoneWhenRecorded(t *testing.T) {

	builder := newAdminStartupBuilder(1,
		form.LookupCode{Label: "Name", Value: "/admin/domain", Href: "/admin/domain/index"},
		form.LookupCode{Label: "Rules", Value: "/admin/rules", Href: "/admin/rules"},
	)

	html := renderAdminStartup(t, "page", builder)
	require.Equal(t, 0, strings.Count(html, "bi-check-circle-fill"))
	require.Equal(t, 2, strings.Count(html, "bi-circle-fill"))
	require.Contains(t, html, `hx-get="/admin/domain/index"`)

	builder.startupTasks = append(builder.startupTasks, "/admin/rules")

	html = renderAdminStartup(t, "page", builder)
	require.Equal(t, 1, strings.Count(html, "bi-check-circle-fill"))
	require.Equal(t, 1, strings.Count(html, "bi-circle-fill"))
}

// TestAdminStartup_TaskIsDoneWhenConnected confirms that a task naming a provider is checked only while that connection is active
func TestAdminStartup_TaskIsDoneWhenConnected(t *testing.T) {

	builder := newAdminStartupBuilder(1,
		form.LookupCode{Label: "Map Tiles", Value: model.ConnectionProviderGeocodeTiles},
	)

	// A connection that was saved but never switched on does not count
	builder.connections[model.ConnectionProviderGeocodeTiles] = model.Connection{Active: false}
	html := renderAdminStartup(t, "page", builder)
	require.Equal(t, 0, strings.Count(html, "bi-check-circle-fill"))

	builder.connections[model.ConnectionProviderGeocodeTiles] = model.Connection{Active: true}
	html = renderAdminStartup(t, "page", builder)
	require.Equal(t, 1, strings.Count(html, "bi-check-circle-fill"))

	// Another provider's connection checks nothing here
	delete(builder.connections, model.ConnectionProviderGeocodeTiles)
	builder.connections[model.ConnectionProviderGiphy] = model.Connection{Active: true}
	html = renderAdminStartup(t, "page", builder)
	require.Equal(t, 0, strings.Count(html, "bi-check-circle-fill"))
}

// TestAdminStartup_PageHearsModalSaves confirms that the checklist redraws when a modal anywhere on the page saves
func TestAdminStartup_PageHearsModalSaves(t *testing.T) {

	html := renderAdminStartup(t, "page", newAdminStartupBuilder(1))
	require.Contains(t, html, `hx-trigger="refreshPage from:window"`)
}

// TestAdminStartup_WritesNoThemeID confirms that no admin-startup action writes themeId, so theme-switch stays the one writer
func TestAdminStartup_WritesNoThemeID(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	adminStartup := templateService.templatePrep["admin-startup"]
	require.NotEmpty(t, adminStartup.Actions)

	for actionID, action := range adminStartup.Actions {
		require.False(t, writesThemeID(action.Steps), "%s must not write themeId", actionID)
	}
}

// writesThemeID returns TRUE if a pipeline, or any pipeline nested in it, has a set-data that reads themeId from the request
func writesThemeID(steps []step.Step) bool {

	stepSlice := reflect.TypeOf([]step.Step{})

	for _, item := range steps {

		if setData, ok := item.(step.SetData); ok {
			if slices.Contains(setData.FromURL, "themeId") || slices.Contains(setData.FromForm, "themeId") {
				return true
			}
		}

		// Search every pipeline that this step holds, such as an as-modal's steps or an if's branches
		value := reflect.Indirect(reflect.ValueOf(item))

		if value.Kind() != reflect.Struct {
			continue
		}

		for index := range value.NumField() {

			if !value.Type().Field(index).IsExported() || (value.Type().Field(index).Type != stepSlice) {
				continue
			}

			if writesThemeID(value.Field(index).Interface().([]step.Step)) {
				return true
			}
		}
	}

	return false
}

// TestWritesThemeID_FindsNestedWriters confirms that the walker reaches a set-data inside a modal inside an if
func TestWritesThemeID_FindsNestedWriters(t *testing.T) {

	parse := func(definition map[string]any) []step.Step {
		parsed, err := step.New(definition, nil)
		require.NoError(t, err)
		return []step.Step{parsed}
	}

	writer := map[string]any{"do": "set-data", "from-url": []any{"themeId"}}
	modal := map[string]any{"do": "as-modal", "steps": []any{writer}}

	require.True(t, writesThemeID(parse(writer)))
	require.True(t, writesThemeID(parse(modal)))
	require.True(t, writesThemeID(parse(map[string]any{"do": "if", "condition": "true", "then": []any{modal}})))
	require.True(t, writesThemeID(parse(map[string]any{"do": "if", "condition": "true", "else": []any{modal}})))
	require.True(t, writesThemeID(parse(map[string]any{"do": "set-data", "from-form": []any{"themeId"}})))

	require.False(t, writesThemeID(parse(map[string]any{"do": "set-data", "from-url": []any{"label"}})))
	require.False(t, writesThemeID(nil))
}
