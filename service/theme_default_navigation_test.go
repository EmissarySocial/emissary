package service

import (
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

/******************************************
 * Default Theme: Mobile Navigation Menu
 *
 * theme-default carries theme-minimal's mobile menu, so these
 * tests hold it to the same structural rules, explained in
 * theme-minimal/AGENTS.md.  theme-default's bar also holds the
 * account links, so the sheet repeats those too.
 ******************************************/

// parseDefaultNavigation renders the real theme-default navigation template and parses the result.
func parseDefaultNavigation(t *testing.T, context minimalNavigationBuilder) *html.Node {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	themeTemplate := template.New("theme-default").Funcs(funcMap)
	require.NoError(t, loadHTMLTemplateFromFilesystem(os.DirFS("../_embed/templates/theme-default"), themeTemplate, funcMap))

	var buffer strings.Builder
	require.NoError(t, themeTemplate.ExecuteTemplate(&buffer, "navigation", context))

	document, err := html.Parse(strings.NewReader(buffer.String()))
	require.NoError(t, err)
	return document
}

// newDefaultNavigationBuilder returns an anonymous visitor on a live domain with two top-level pages.
func newDefaultNavigationBuilder() minimalNavigationBuilder {
	return minimalNavigationBuilder{
		DomainStateID:             model.DomainStateLive,
		DomainHasRegistrationForm: true,
		themeData:                 mapof.NewAny(),
		Navigation: []minimalNavigationItem{
			{StreamID: "6a908f1d331aab271b96e5f8", Token: "about", Label: "About Me"},
			{StreamID: "6a9484676097f8f87ef9a455", Token: "contact", Label: "Contact Me"},
		},
	}
}

// TestDefaultTheme_PageDoesNotWrapNavigation keeps the menu out of <nav>: page.html must not wrap
// the template, or the sheet inherits every `nav` and `nav a` rule.
func TestDefaultTheme_PageDoesNotWrapNavigation(t *testing.T) {

	page, err := os.ReadFile("../_embed/templates/theme-default/page.html")
	require.NoError(t, err)

	require.NotContains(t, string(page), "<nav")
	require.Contains(t, string(page), `{{- template "navigation" . -}}`)
}

// TestDefaultTheme_SkipLinkOutsideBothLayouts keeps the skip link reachable while either layout is hidden.
func TestDefaultTheme_SkipLinkOutsideBothLayouts(t *testing.T) {

	document := parseDefaultNavigation(t, newDefaultNavigationBuilder())

	skipLink := navFindOne(t, document, "skip link", func(node *html.Node) bool {
		return node.Data == "a" && navAttribute(node, "href") == "#main"
	})

	require.False(t, navHasAncestor(skipLink, navIsTag("nav")))
	require.False(t, navHasAncestor(skipLink, navHasID("nav-menu-panel")))
}

// TestDefaultTheme_MenuIsNotASecondNavElement keeps exactly one <nav>, with the menu as a named landmark beside it.
func TestDefaultTheme_MenuIsNotASecondNavElement(t *testing.T) {

	document := parseDefaultNavigation(t, newDefaultNavigationBuilder())

	require.Len(t, navFind(document, navIsTag("nav")), 1)

	menu := navFindOne(t, document, "mobile menu wrapper", func(node *html.Node) bool {
		return navHasClass(node, "nav-menu")
	})

	require.Equal(t, "navigation", navAttribute(menu, "role"))
	require.NotEmpty(t, navAttribute(menu, "aria-label"))
	require.False(t, navHasAncestor(menu, navIsTag("nav")), "the menu must sit beside <nav>, not inside it")

	bar := navFindOne(t, document, "<nav> element", navIsTag("nav"))
	navFindOne(t, bar, "wrapper carrying class=framed inside <nav>", func(node *html.Node) bool {
		return navHasClass(node, "framed")
	})
}

// TestDefaultTheme_ToggleSiblingOrder pins the order `+` and `~` depend on: checkbox, label, then sheet.
func TestDefaultTheme_ToggleSiblingOrder(t *testing.T) {

	document := parseDefaultNavigation(t, newDefaultNavigationBuilder())

	toggle := navFindOne(t, document, "menu checkbox", navHasID("nav-menu-toggle"))
	label := navNextElement(toggle)

	require.NotNil(t, label)
	require.Equal(t, "label", label.Data)
	require.Equal(t, "nav-menu-toggle", navAttribute(label, "for"))
	require.Empty(t, navAttribute(label, "role"), "role on the label would cost a second tab stop")

	require.Equal(t, "button", navAttribute(toggle, "role"))
	require.Equal(t, "off", navAttribute(toggle, "autocomplete"))
	require.True(t, navHasClass(toggle, "hide-visual"))

	panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))
	require.Same(t, toggle.Parent, panel.Parent)
	require.Same(t, label, navNextElement(toggle))
	require.Same(t, panel, navNextElement(label), "the sheet follows the label")
}

// TestDefaultTheme_SheetCopiesAreInvisibleToSelectNav keeps ids and .nav-item on the bar's links only.
func TestDefaultTheme_SheetCopiesAreInvisibleToSelectNav(t *testing.T) {

	for _, context := range defaultNavigationVisitors() {

		document := parseDefaultNavigation(t, context)
		panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))

		links := navFind(panel, navIsTag("a"))
		require.NotEmpty(t, links)

		for _, link := range links {
			require.Empty(t, navAttribute(link, "id"), navAttribute(link, "href"))
			require.False(t, navHasClass(link, "nav-item"), navAttribute(link, "href"))
			require.True(t, navHasClass(link, "turboclick"), navAttribute(link, "href"))
		}
	}
}

// TestDefaultTheme_EveryLinkRenderedTwice holds the bar and the sheet to the same links, for every kind of visitor.
func TestDefaultTheme_EveryLinkRenderedTwice(t *testing.T) {

	expected := [][]string{
		{"/about", "/contact", "/signin", "/register"},
		{"/about", "/contact", "/@me", "/@me/newsfeed", "/@me/conversations", "/@me/notifications/replies", "/admin/domain"},
		{"/about", "/contact", "/@guest"},
	}

	for index, context := range defaultNavigationVisitors() {

		counts := mapof.NewInt()

		for _, link := range navFind(parseDefaultNavigation(t, context), navIsTag("a")) {
			counts[navAttribute(link, "href")]++
		}

		for _, href := range expected[index] {
			require.Equal(t, 2, counts[href], "%s must render once in the bar and once in the sheet", href)
		}

		require.Equal(t, 1, counts["#main"])
	}
}

// TestDefaultTheme_SheetMarksCurrentSection marks the current section server-side, and nothing for an unmatched id.
func TestDefaultTheme_SheetMarksCurrentSection(t *testing.T) {

	selected := func(navigationID string) []string {

		context := newDefaultNavigationBuilder()
		context.NavigationID = navigationID

		panel := navFindOne(t, parseDefaultNavigation(t, context), "menu sheet", navHasID("nav-menu-panel"))

		result := []string{}
		for _, link := range navFind(panel, navIsTag("a")) {
			if navHasClass(link, "selected") {
				require.Equal(t, "page", navAttribute(link, "aria-current"))
				result = append(result, navAttribute(link, "href"))
			}
		}
		return result
	}

	require.Equal(t, []string{"/contact"}, selected("6a9484676097f8f87ef9a455"))
	require.Empty(t, selected(""))
	require.Empty(t, selected("ffffffffffffffffffffffff"))
}

// TestDefaultTheme_StartupReachesTheSheet shows the Startup Checklist link in the sheet as well as the bar.
func TestDefaultTheme_StartupReachesTheSheet(t *testing.T) {

	context := newDefaultNavigationBuilder()
	context.DomainStateID = model.DomainStateStartup

	panel := navFindOne(t, parseDefaultNavigation(t, context), "menu sheet", navHasID("nav-menu-panel"))

	navFindOne(t, panel, "startup link inside the sheet", func(node *html.Node) bool {
		return node.Data == "a" && navAttribute(node, "href") == "/startup" && navHasClass(node, "turboclick")
	})
}

// TestDefaultTheme_BarHidesOnlyBesideTheMenu pins the scoped hide rule: qwertylicious loads this
// stylesheet, and its desktop sidebar is a <nav> with no .nav-menu beside it.
func TestDefaultTheme_BarHidesOnlyBesideTheMenu(t *testing.T) {

	stylesheet, err := os.ReadFile("../_embed/templates/theme-default/stylesheet/04-nav-menu.css")
	require.NoError(t, err)

	require.Contains(t, string(stylesheet), "body:has(> .nav-menu) > nav {")
	require.NotContains(t, string(stylesheet), "\tnav {", "a bare `nav` rule would hide every theme's bar")
}

// defaultNavigationVisitors returns an anonymous visitor, a signed-in owner, and a guest identity.
func defaultNavigationVisitors() []minimalNavigationBuilder {

	anonymous := newDefaultNavigationBuilder()

	owner := newDefaultNavigationBuilder()
	owner.IsAuthenticated = true
	owner.IsOwner = true

	guest := newDefaultNavigationBuilder()
	guest.IsIdentity = true

	return []minimalNavigationBuilder{anonymous, owner, guest}
}
