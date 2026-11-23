package service

import (
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

/******************************************
 * Default Theme: Mobile Navigation Menu
 *
 * theme-default renders its links twice: once in the desktop
 * bar, and once in the mobile menu's sheet.  These tests hold
 * the two to the structural rules that keep them apart.  The
 * bar also holds the account links, so the sheet repeats those.
 ******************************************/

// navItem stands in for a row of build.Common.Navigation: the
// top-level Streams that the theme lists in its navigation bar.
type navItem struct {
	StreamID string
	Token    string
	Label    string
	Icon     string
}

// navBuilder stands in for the build.Builder that renders the
// theme's page chrome, exposing only the fields navigation.html reads.
type navBuilder struct {
	DomainStateID             string
	Navigation                []navItem
	NavigationID              string
	IsAuthenticated           bool
	IsIdentity                bool
	IsOwner                   bool
	HasUnreadNotifications    bool
	DomainHasRegistrationForm bool
	themeData                 mapof.Any
	themeSchema               schema.Schema
}

// NotAuthenticatedOrIdentity mirrors build.Common.NotAuthenticatedOrIdentity: TRUE for a
// visitor who is neither a signed-in user nor a guest identity.
func (builder navBuilder) NotAuthenticatedOrIdentity() bool {
	return !builder.IsAuthenticated && !builder.IsIdentity
}

// ThemeData mirrors build.Common.ThemeData: the stored value coerced to a string, falling
// back to the Theme schema's declared default for a token the Domain has never saved.
func (builder navBuilder) ThemeData(token string) string {

	if value, exists := builder.themeData[token]; exists {
		return convert.String(value)
	}

	element, exists := builder.themeSchema.GetElement("themeData." + token)

	if !exists {
		return ""
	}

	return convert.String(element.DefaultValue())
}

// parseDefaultNavigation renders the real theme-default navigation template and parses the result.
func parseDefaultNavigation(t *testing.T, context navBuilder) *html.Node {

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
func newDefaultNavigationBuilder() navBuilder {
	return navBuilder{
		DomainStateID:             model.DomainStateLive,
		DomainHasRegistrationForm: true,
		themeData:                 mapof.NewAny(),
		Navigation: []navItem{
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
func defaultNavigationVisitors() []navBuilder {

	anonymous := newDefaultNavigationBuilder()

	owner := newDefaultNavigationBuilder()
	owner.IsAuthenticated = true
	owner.IsOwner = true

	guest := newDefaultNavigationBuilder()
	guest.IsIdentity = true

	return []navBuilder{anonymous, owner, guest}
}

// navAttribute returns one attribute's value, or "" when the node does not carry it.
func navAttribute(node *html.Node, name string) string {

	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}

	return ""
}

// navHasClass reports whether the node carries one class TOKEN.  Token matching is the whole
// point: "nav-menu-item" must not answer to "nav-item", because that is exactly how CSS and
// hyperscript match, and it is what keeps SelectNav away from the sheet's copies.
func navHasClass(node *html.Node, class string) bool {

	for _, candidate := range strings.Fields(navAttribute(node, "class")) {
		if candidate == class {
			return true
		}
	}

	return false
}

// navFind returns every element matching the predicate, in document order.
func navFind(root *html.Node, match func(*html.Node) bool) []*html.Node {

	var result []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {

		if node.Type == html.ElementNode {
			if match(node) {
				result = append(result, node)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return result
}

// navFindOne returns the single element matching the predicate, failing when there is not
// exactly one.
func navFindOne(t *testing.T, root *html.Node, description string, match func(*html.Node) bool) *html.Node {

	t.Helper()

	found := navFind(root, match)
	require.Len(t, found, 1, "expected exactly one %s", description)

	return found[0]
}

// navHasAncestor reports whether the node is a DESCENDANT of an element matching the
// predicate -- the question a substring assertion cannot answer.
func navHasAncestor(node *html.Node, match func(*html.Node) bool) bool {

	for parent := node.Parent; parent != nil; parent = parent.Parent {

		if parent.Type == html.ElementNode {
			if match(parent) {
				return true
			}
		}
	}

	return false
}

// navNextElement returns the next SIBLING element, skipping the whitespace text nodes
// between them -- which is what the CSS adjacent sibling combinator does too.
func navNextElement(node *html.Node) *html.Node {

	for sibling := node.NextSibling; sibling != nil; sibling = sibling.NextSibling {

		if sibling.Type == html.ElementNode {
			return sibling
		}
	}

	return nil
}

// navIsTag matches an element by tag name.
func navIsTag(name string) func(*html.Node) bool {
	return func(node *html.Node) bool {
		return node.Data == name
	}
}

// navHasID matches an element by its id attribute.
func navHasID(id string) func(*html.Node) bool {
	return func(node *html.Node) bool {
		return navAttribute(node, "id") == id
	}
}
