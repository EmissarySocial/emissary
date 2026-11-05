package service

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// adminThemesBuilder is the smallest dot that the admin-domain theme pages render against
type adminThemesBuilder struct {
	themeID string                 // The Domain's stored ThemeID
	query   string                 // The ?themeId= query parameter
	themes  map[string]model.Theme // Every installed Theme, visible or not
}

// newAdminThemesBuilder returns a stub with two visible Themes and one hidden one, like a real server
func newAdminThemesBuilder(themeID string, query string) adminThemesBuilder {

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	themes := make(map[string]model.Theme)

	for _, item := range []struct{ id, label string }{{"default", "Default"}, {"minimal", "Minimal"}, {"global", "Global"}} {
		theme := model.NewTheme(item.id, funcMap)
		theme.Label = item.label
		theme.Description = item.label + " description"
		theme.IsVisible = (item.id != "global")
		themes[item.id] = theme
	}

	return adminThemesBuilder{themeID: themeID, query: query, themes: themes}
}

// ThemeID returns the Domain's stored ThemeID
func (builder adminThemesBuilder) ThemeID() string {
	return builder.themeID
}

// Theme returns the named Theme, or the default Theme for an unknown ID, as service.Theme.GetTheme does
func (builder adminThemesBuilder) Theme(themeID string) model.Theme {
	if theme, ok := builder.themes[themeID]; ok {
		return theme
	}
	return builder.themes["default"]
}

// Themes returns the visible Themes in display order, as build.Domain.Themes does
func (builder adminThemesBuilder) Themes() []model.Theme {
	return []model.Theme{builder.themes["default"], builder.themes["minimal"]}
}

// QueryParam returns the stub's ?themeId= value, and an empty string for every other parameter
func (builder adminThemesBuilder) QueryParam(param string) string {
	if param == "themeId" {
		return builder.query
	}
	return ""
}

// adminSettingsBuilder adds what the admin menubar calls, so that the settings page itself can render
type adminSettingsBuilder struct {
	adminThemesBuilder
}

// Token returns the admin section that the menubar marks as open
func (builder adminSettingsBuilder) Token() string {
	return "domain"
}

// IsDesktop returns TRUE, so that the menubar draws its sidebar
func (builder adminSettingsBuilder) IsDesktop() bool {
	return true
}

// IsMobile returns FALSE, so that the menubar draws no back arrow
func (builder adminSettingsBuilder) IsMobile() bool {
	return false
}

// IsDomainStartup returns FALSE, so that no startup banner is drawn
func (builder adminSettingsBuilder) IsDomainStartup() bool {
	return false
}

// renderAdminDomain executes one admin-domain HTML template against the stub builder
func renderAdminDomain(t *testing.T, templateService *Template, file string, builder any) string {

	t.Helper()

	var buffer bytes.Buffer
	adminDomain := templateService.templatePrep["admin-domain"]
	require.NoError(t, adminDomain.HTMLTemplate.ExecuteTemplate(&buffer, file, builder))

	return buffer.String()
}

// TestAdminDomainThemes_SettingsCard confirms that the settings page shows the current theme, and opens the picker
func TestAdminDomainThemes_SettingsCard(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)

	html := renderAdminDomain(t, templateService, "index", adminSettingsBuilder{newAdminThemesBuilder("minimal", "")})
	require.Contains(t, html, "Minimal description")
	require.Contains(t, html, `hx-get="/admin/domain/themes"`)
	require.NotContains(t, html, "hx-post", "the card changes nothing by itself")

	// A Domain that has never chosen a theme shows the default one
	html = renderAdminDomain(t, templateService, "index", adminSettingsBuilder{newAdminThemesBuilder("", "")})
	require.Contains(t, html, "Default description")
}

// TestAdminDomainThemes_PickerSavesNothing confirms that the picker marks the current theme and only links to theme-confirm
func TestAdminDomainThemes_PickerSavesNothing(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	html := renderAdminDomain(t, templateService, "themes", newAdminThemesBuilder("minimal", ""))

	require.Equal(t, 1, strings.Count(html, `aria-current="true"`))
	require.Contains(t, html, `style="pointer-events:none" aria-current="true"`, "the current theme is not interactive")
	require.Contains(t, html, `hx-get="/admin/domain/theme-confirm?themeId=default"`)
	require.NotContains(t, html, " description", "the picker lists names only")
	require.NotContains(t, html, `themeId=minimal`, "the current theme cannot be chosen again")
	require.NotContains(t, html, `themeId=global`, "a hidden theme is never offered")
	require.NotContains(t, html, `hx-post`, "choosing a theme saves nothing")

	// A Domain that has never chosen a theme is using the default one, so that one is current
	html = renderAdminDomain(t, templateService, "themes", newAdminThemesBuilder("", ""))
	require.NotContains(t, html, `themeId=default`)
	require.Contains(t, html, `hx-get="/admin/domain/theme-confirm?themeId=minimal"`)
}

// TestAdminDomainThemes_OnlyThemeSwitchSaves confirms that theme-switch is the one admin-domain action that writes themeId
func TestAdminDomainThemes_OnlyThemeSwitchSaves(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	adminDomain := templateService.templatePrep["admin-domain"]

	// The two modal steps only draw a view
	for _, actionID := range []string{"themes", "theme-confirm"} {

		action := adminDomain.Actions[actionID]
		require.Len(t, action.Steps, 1, actionID)

		modal, ok := action.Steps[0].(step.AsModal)
		require.True(t, ok, "%s must be an as-modal, not %T", actionID, action.Steps[0])
		require.Len(t, modal.SubSteps, 1, actionID)

		view, ok := modal.SubSteps[0].(step.ViewHTML)
		require.True(t, ok, "%s must only draw a view, not %T", actionID, modal.SubSteps[0])
		require.Equal(t, "get", view.Method, actionID)
	}

	// theme-switch saves only when its condition passes, and redraws theme-confirm when it fails
	action := adminDomain.Actions["theme-switch"]
	require.Len(t, action.Steps, 1)

	condition, ok := action.Steps[0].(step.IfCondition)
	require.True(t, ok, "theme-switch must be an if, not %T", action.Steps[0])

	setData, ok := condition.Then[0].(step.SetData)
	require.True(t, ok, "theme-switch must start with set-data, not %T", condition.Then[0])
	require.Equal(t, []string{"themeId"}, setData.FromURL)
	require.Empty(t, setData.FromForm)

	require.Len(t, condition.Otherwise, 1)
	refusal, ok := condition.Otherwise[0].(step.ViewHTML)
	require.True(t, ok, "a refusal must redraw theme-confirm, not %T", condition.Otherwise[0])
	require.Equal(t, "theme-confirm", refusal.File)
	require.Equal(t, "post", refusal.Method)
}

// TestAdminDomainThemes_ConfirmMatchesSwitch confirms that theme-confirm offers the switch exactly when theme-switch allows it
func TestAdminDomainThemes_ConfirmMatchesSwitch(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	adminDomain := templateService.templatePrep["admin-domain"]
	condition := adminDomain.Actions["theme-switch"].Steps[0].(step.IfCondition)

	tests := []struct {
		stored    string
		requested string
		allowed   bool
	}{
		{stored: "default", requested: "minimal", allowed: true},
		{stored: "minimal", requested: "default", allowed: true},
		{stored: "", requested: "minimal", allowed: true},
		{stored: "default", requested: "default", allowed: false},
		{stored: "", requested: "default", allowed: false},
		{stored: "removed", requested: "default", allowed: false},
		{stored: "default", requested: "global", allowed: false},
		{stored: "default", requested: "unknown", allowed: false},
		{stored: "default", requested: "", allowed: false},
		{stored: "default", requested: "Minimal", allowed: false},
		{stored: "default", requested: " minimal", allowed: false},
		{stored: "default", requested: `minimal"><script>`, allowed: false},
	}

	for _, test := range tests {

		builder := newAdminThemesBuilder(test.stored, test.requested)

		// RULE: The pipeline's condition and the view must give the same answer
		var value bytes.Buffer
		require.NoError(t, condition.Condition.Execute(&value, builder))
		require.Equal(t, test.allowed, value.String() == "true", "stored %q, requested %q", test.stored, test.requested)

		html := renderAdminDomain(t, templateService, "theme-confirm", builder)

		// Every answer offers the way back to the picker
		require.Contains(t, html, `hx-get="/admin/domain/themes"`, "stored %q, requested %q", test.stored, test.requested)

		if test.allowed {
			next := builder.Theme(test.requested)
			require.Contains(t, html, `hx-post="/admin/domain/theme-switch?themeId=`+test.requested+`"`, "stored %q, requested %q", test.stored, test.requested)
			require.Contains(t, html, `hx-target="#modal-window"`)
			require.Regexp(t, `<h1 id="modal-title">[^<]*`+regexp.QuoteMeta(next.Label)+`</h1>`, html, "the heading names the theme, after its icon")
			require.Contains(t, html, `src="/.themes/`+test.requested+`/resources/startup/splash.png"`)
			require.Contains(t, html, next.Description)
			require.NotContains(t, html, builder.Theme(test.stored).Label, "the current theme is not shown")
			continue
		}

		require.NotContains(t, html, "hx-post", "stored %q, requested %q", test.stored, test.requested)
		require.Contains(t, html, "Theme Not Available", "stored %q, requested %q", test.stored, test.requested)
	}
}
