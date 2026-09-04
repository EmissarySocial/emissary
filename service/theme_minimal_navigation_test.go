package service

import (
	"html/template"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

/******************************************
 * Minimal Theme: Navigation Toggles
 *
 * The "minimal" theme lets a Domain owner hide the top-level
 * navigation from the theme settings form.  These tests run the
 * REAL theme.hjson and the REAL navigation.html, because the two
 * halves are only correct together: the form writes a boolean
 * into Domain.ThemeData, and the template reads it back through
 * .ThemeData, which stringifies it.
 *
 * The trap they exist to pin down is the DEFAULT.  Domain.ThemeData
 * starts out empty, an absent boolean reads as FALSE, and the theme
 * form posts every tab at once -- so without the "default" declared
 * on these properties, the toggles would render OFF and the first
 * save of ANY tab would write that phantom FALSE and blank the
 * navigation bar.  Both the form widget and build.Common.ThemeData
 * fall back to that one declared default, and these tests hold the
 * two of them to it.
 ******************************************/

// minimalNavigationItem stands in for a row of build.Common.Navigation: the
// top-level Streams that the theme lists in its navigation bar.
type minimalNavigationItem struct {
	StreamID string
	Token    string
	Label    string
	Icon     string
}

// minimalNavigationBuilder stands in for the build.Builder that renders the
// theme's page chrome, exposing only the fields navigation.html reads.
type minimalNavigationBuilder struct {
	DomainStateID             string
	Navigation                []minimalNavigationItem
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
func (builder minimalNavigationBuilder) NotAuthenticatedOrIdentity() bool {
	return !builder.IsAuthenticated && !builder.IsIdentity
}

// ThemeData mirrors build.Common.ThemeData: the stored value coerced to a string, falling
// back to the Theme schema's declared default for a token the Domain has never saved.
func (builder minimalNavigationBuilder) ThemeData(token string) string {

	if value, exists := builder.themeData[token]; exists {
		return convert.String(value)
	}

	element, exists := builder.themeSchema.GetElement("themeData." + token)

	if !exists {
		return ""
	}

	return convert.String(element.DefaultValue())
}

// loadMinimalTheme parses the real theme-minimal definition the same way service.Theme.Add does
func loadMinimalTheme(t *testing.T) model.Theme {

	t.Helper()

	filesystem := os.DirFS("../_embed/templates/theme-minimal")
	definitionType, definition := findDefinition(filesystem)
	require.Equal(t, DefinitionTheme, definitionType)

	theme := model.NewTheme("minimal", emissarytemplates.FuncMap(nullIconProvider{}))
	require.NoError(t, hjson.Unmarshal(definition, &theme))
	require.NoError(t, theme.Schema.ValidateFormats())

	return theme
}

// minimalDomainSchema rebuilds the merged Domain schema exactly as build.Domain.schema() does,
// so that the theme's themeData properties layer onto the Domain's own wildcard object.
func minimalDomainSchema(t *testing.T) schema.Schema {

	t.Helper()

	result := schema.New(model.DomainSchema())
	result.Inherit(loadMinimalTheme(t).Schema)
	return result
}

// renderMinimalNavigation parses the real theme-minimal folder the way the server does at
// startup, then executes its navigation template against the provided context.
func renderMinimalNavigation(t *testing.T, context minimalNavigationBuilder) string {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	themeTemplate := template.New("theme-minimal").Funcs(funcMap)

	filesystem := os.DirFS("../_embed/templates/theme-minimal")
	require.NoError(t, loadHTMLTemplateFromFilesystem(filesystem, themeTemplate, funcMap))

	var buffer strings.Builder
	require.NoError(t, themeTemplate.ExecuteTemplate(&buffer, "navigation", context))

	return buffer.String()
}

// newMinimalNavigationBuilder returns an anonymous visitor on a LIVE domain that publishes
// two top-level pages -- the shape in which the navigation bar is visible by default.
func newMinimalNavigationBuilder(t *testing.T, themeData mapof.Any) minimalNavigationBuilder {

	t.Helper()

	return minimalNavigationBuilder{
		DomainStateID: model.DomainStateLive,
		themeData:     themeData,
		themeSchema:   loadMinimalTheme(t).Schema,
		Navigation: []minimalNavigationItem{
			{StreamID: "6a908f1d331aab271b96e5f8", Token: "about", Label: "About Me"},
			{StreamID: "6a9484676097f8f87ef9a455", Token: "contact", Label: "Contact Me"},
		},
	}
}

// TestMinimalTheme_NavigationToggle_Default verifies that a Domain whose owner has never
// touched the toggle still gets its navigation bar.  This is the pristine case: ThemeData
// is empty, so the flag resolves through the Theme schema's declared default.
func TestMinimalTheme_NavigationToggle_Default(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)
	require.Contains(t, result, `href="/signin"`)
	require.Contains(t, result, `href="/register"`)
}

// TestMinimalTheme_NavigationToggle_Visible verifies the toggle in its ON position, which is
// also what the form posts for every save that leaves the toggle alone.
func TestMinimalTheme_NavigationToggle_Visible(t *testing.T) {

	result := renderMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.Any{"showNavigation": true}))

	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)
}

// TestMinimalTheme_NavigationToggle_Hidden verifies that switching the toggle OFF drops the
// whole navigation bar.  The flag wraps the entire <nav>, not just the top-level links, so
// the guest Sign In block goes with it.
func TestMinimalTheme_NavigationToggle_Hidden(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.Any{"showNavigation": false})
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.NotContains(t, result, "<nav>")
	require.NotContains(t, result, `href="/about"`)
	require.NotContains(t, result, `href="/signin"`)
}

// TestMinimalTheme_SignupToggle_Hidden verifies that the Sign In toggle drops the guest
// call-to-action WITHOUT taking the rest of the navigation bar with it.  The two flags gate
// different amounts of the bar, so they need to be pinned apart.
func TestMinimalTheme_SignupToggle_Hidden(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.Any{"showSignup": false})
	context.DomainHasRegistrationForm = true

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/about"`)
	require.NotContains(t, result, `href="/signin"`)
	require.NotContains(t, result, `href="/register"`)
}

// TestMinimalTheme_NavigationToggle_Startup verifies that a domain still in STARTUP reaches
// its Startup Checklist, which is the owner's path out of that state.  A domain in STARTUP
// has never saved its settings, so the flag resolves through the default here.
func TestMinimalTheme_NavigationToggle_Startup(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainStateID = model.DomainStateStartup

	result := renderMinimalNavigation(t, context)

	require.Contains(t, result, `href="/startup"`)
}

// TestMinimalTheme_SignedInShortcuts verifies that the floating menu renders outside <nav>,
// and keeps doing so when the navigation bar itself is switched off -- the two are
// independent, and a user who hides the bar must not lose their own links.
func TestMinimalTheme_SignedInShortcuts(t *testing.T) {

	for _, themeData := range []mapof.Any{
		mapof.NewAny(),
		{"showNavigation": false},
	} {
		context := newMinimalNavigationBuilder(t, themeData)
		context.IsAuthenticated = true

		result := renderMinimalNavigation(t, context)

		require.Contains(t, result, `class="floating-menu card padding-xs"`)
		require.Contains(t, result, `href="/@me"`)
		require.Contains(t, result, `href="/@me/notifications/replies"`)
	}
}

// TestMinimalTheme_FloatingMenuClassIsShared verifies that BOTH signed-in branches carry the
// same self-contained class.  The guest-identity branch used to put the positioning and the
// card on one element while the authenticated branch nested them, so a selector written for
// one shape silently missed the other -- and a missed .floating-menu is a collapsed menu.
func TestMinimalTheme_FloatingMenuClassIsShared(t *testing.T) {

	for name, setup := range map[string]func(*minimalNavigationBuilder){
		"authenticated": func(b *minimalNavigationBuilder) { b.IsAuthenticated = true },
		"identity":      func(b *minimalNavigationBuilder) { b.IsIdentity = true },
	} {
		context := newMinimalNavigationBuilder(t, mapof.NewAny())
		setup(&context)

		result := renderMinimalNavigation(t, context)

		require.Contains(t, result, `class="floating-menu card padding-xs"`, name)

		// The old two-element shape must not come back: .floating-menu carries its own
		// positioning, so pairing it with the positioning utility would double up.
		require.NotContains(t, result, `pos-absolute-bottom-right`, name)
	}
}

// TestMinimalTheme_Form_Toggles verifies that both toggles reach the rendered settings form,
// bound to the themeData properties the theme's schema declares.  A form field naming a
// property that the schema does not define renders, but silently drops its value on save.
func TestMinimalTheme_Form_Toggles(t *testing.T) {

	widget.UseAll()

	domainSchema := minimalDomainSchema(t)

	for _, path := range []string{"themeData.showNavigation", "themeData.showSignup"} {

		element, exists := domainSchema.GetElement(path)
		require.True(t, exists, "schema does not declare %s", path)
		require.IsType(t, schema.Boolean{}, element, "%s must be a boolean", path)
	}

	domain := model.NewDomain()
	settingsForm := form.Form{Schema: domainSchema, Element: loadMinimalTheme(t).Form}

	result, err := settingsForm.Editor(&domain, nil)
	require.NoError(t, err)

	require.Contains(t, result, `name="themeData.showNavigation"`)
	require.Contains(t, result, `name="themeData.showSignup"`)

	// Both toggles render ON for a Domain that has never saved them.  The toggle posts
	// whatever it displays, and the tab layout posts every field on every save, so a toggle
	// that displayed OFF here would write FALSE the first time the owner saved any tab.
	require.Equal(t, 2, strings.Count(result, `value="true"`))
}

// TestMinimalTheme_Form_SavePreservesNavigation is the polarity guard.  The settings form is a
// tab layout, so saving ANY tab posts EVERY field, including both toggles.  Saving a pristine
// Domain must therefore leave the navigation visible -- if the flags were ever renamed to
// "show" polarity, this is the test that would catch the blanked navigation bar.
func TestMinimalTheme_Form_SavePreservesNavigation(t *testing.T) {

	widget.UseAll()

	domain := model.NewDomain()
	settingsForm := form.Form{Schema: minimalDomainSchema(t), Element: loadMinimalTheme(t).Form}

	// A toggle posts whatever it displays, and both of these display ON, so this is what
	// the browser sends for a save that leaves them alone
	values := url.Values{
		"label":                    []string{"My Site"},
		"themeData.showNavigation": []string{"true"},
		"themeData.showSignup":     []string{"true"},
	}

	require.NoError(t, settingsForm.SetURLValues(&domain, values, nil))

	// It round-trips as a real boolean, not the string "true"
	require.Equal(t, true, domain.ThemeData["showNavigation"])

	result := renderMinimalNavigation(t, newMinimalNavigationBuilder(t, domain.ThemeData))
	require.Contains(t, result, `href="/about"`)

	// And switching it off still works -- the default must not override a stored FALSE
	values.Set("themeData.showNavigation", "false")
	require.NoError(t, settingsForm.SetURLValues(&domain, values, nil))
	require.Equal(t, false, domain.ThemeData["showNavigation"])

	result = renderMinimalNavigation(t, newMinimalNavigationBuilder(t, domain.ThemeData))
	require.NotContains(t, result, `href="/about"`)
}

/******************************************
 * Minimal Theme: Mobile Navigation Menu
 *
 * Below 640px the horizontal bar is display:none and every site
 * navigation link is reached through a fixed control in the
 * top-right corner that opens a full-screen sheet.  The bar and
 * the sheet are SEPARATE elements rendering the SAME links twice.
 *
 * Every claim these tests make is about ancestry, sibling order,
 * or the presence of one attribute -- never about text -- so they
 * parse the rendered markup instead of matching substrings.  A
 * substring assertion cannot tell "inside <nav>" from "beside it",
 * and that distinction is the entire feature: the first attempt at
 * this menu reached inside <nav>, removed `class="framed"` from the
 * bar's wrapper, and took the desktop bar's flex layout, centering
 * and 1100px frame with it.  Nothing rendered an error.
 ******************************************/

// parseMinimalNavigation renders the real navigation template and parses the result.
func parseMinimalNavigation(t *testing.T, context minimalNavigationBuilder) *html.Node {

	t.Helper()

	document, err := html.Parse(strings.NewReader(renderMinimalNavigation(t, context)))
	require.NoError(t, err)

	return document
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

// TestMinimalTheme_SkipLinkOutsideBothLayouts is the reason the skip link moved.  It used to be
// the first child of the .framed div; <nav> is now display:none below 640px, and display:none
// removes an element from the accessibility tree -- so a skip link left inside would stop
// existing on every mobile page, for exactly the visitors it is there for.  The sheet is no
// refuge either: it is display:none whenever the menu is closed, which is always on load.
func TestMinimalTheme_SkipLinkOutsideBothLayouts(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.NewAny()))

	skipLink := navFindOne(t, document, "skip link", func(node *html.Node) bool {
		return node.Data == "a" && navAttribute(node, "href") == "#main"
	})

	require.False(t, navHasAncestor(skipLink, navIsTag("nav")),
		"the skip link must not be inside <nav>, which is display:none below 640px")

	require.False(t, navHasAncestor(skipLink, navHasID("nav-menu-panel")),
		"the skip link must not be inside the sheet, which is display:none while the menu is closed")
}

// TestMinimalTheme_BarWrapperKeepsFramedClass pins down the class whose removal broke the first
// attempt.  `.framed` is not a nav class: it is the shared page-width class from 02-page.css,
// and inside <nav> it is additionally what supplies the bar's flex layout, its centering at
// >=640px, and its 1100px frame at >=1024px.  Losing it costs all of that silently.
func TestMinimalTheme_BarWrapperKeepsFramedClass(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.NewAny()))

	bar := navFindOne(t, document, "<nav> element", navIsTag("nav"))

	navFindOne(t, bar, "wrapper carrying class=framed inside <nav>", func(node *html.Node) bool {
		return navHasClass(node, "framed")
	})
}

// TestMinimalTheme_MenuIsNotASecondNavElement guards the landmark decision.  The theme's nav
// styling is ELEMENT-scoped -- `nav { height:48px; overflow-x:auto; … }` and `nav a { … }` --
// so a second <nav> would silently inherit the whole 48px bar treatment.  The sheet carries a
// role instead, on a wrapper that stays in the tree while the menu is closed.
func TestMinimalTheme_MenuIsNotASecondNavElement(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.NewAny()))

	require.Len(t, navFind(document, navIsTag("nav")), 1, "there must be exactly one <nav> element")

	menu := navFindOne(t, document, "mobile menu wrapper", func(node *html.Node) bool {
		return navHasClass(node, "nav-menu")
	})

	require.Equal(t, "navigation", navAttribute(menu, "role"))
	require.NotEmpty(t, navAttribute(menu, "aria-label"),
		"two navigation landmarks on one page are indistinguishable in a landmark list without names")

	// The wrapper stays in the tree when the menu is closed, so the panel cannot be the
	// landmark -- it is display:none in that state.
	require.False(t, navHasClass(menu, "nav-menu-panel"))
}

// TestMinimalTheme_ToggleSiblingOrder pins the DOM order the CSS depends on.  The label is
// reached with `+` and the sheet with `~`, and both combinators only look FORWARD -- so a
// checkbox emitted after either one leaves the menu permanently closed with no error anywhere.
func TestMinimalTheme_ToggleSiblingOrder(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.NewAny()))

	toggle := navFindOne(t, document, "menu checkbox", navHasID("nav-menu-toggle"))
	label := navNextElement(toggle)

	require.NotNil(t, label, "the checkbox must have a following sibling")
	require.Equal(t, "label", label.Data, "the label must be the checkbox's ADJACENT sibling, for `+`")
	require.Equal(t, "nav-menu-toggle", navAttribute(label, "for"))

	// And the sheet is a LATER sibling of the same parent, for `~`
	panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))
	require.Same(t, toggle.Parent, panel.Parent, "the checkbox and the sheet must be siblings")

	var seenToggle bool
	var seenPanel bool

	for sibling := toggle.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		switch sibling {
		case toggle:
			seenToggle = true
		case panel:
			seenPanel = true
			require.True(t, seenToggle, "the checkbox must come BEFORE the sheet, for `~`")
		}
	}

	require.True(t, seenPanel)
}

// TestMinimalTheme_ToggleCarriesRoleAndLabelDoesNot is the a11y-extension trap.  a11y.js walks
// `a,button,[role=link],[role=button],[role=tab]`, sets tabIndex=0 on anything at -1, and adds
// an Enter handler.  On the checkbox that is exactly right: it is already focusable, so the
// tabIndex line is a no-op, and Enter is the key a native checkbox does NOT answer.  On the
// <label> it is a trap -- a label sits at tabIndex -1, so this one control would take two tab
// stops.
func TestMinimalTheme_ToggleCarriesRoleAndLabelDoesNot(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.NewAny()))

	toggle := navFindOne(t, document, "menu checkbox", navHasID("nav-menu-toggle"))

	require.Equal(t, "checkbox", navAttribute(toggle, "type"))
	require.Equal(t, "button", navAttribute(toggle, "role"))
	require.Equal(t, "nav-menu-panel", navAttribute(toggle, "aria-controls"))
	require.NotEmpty(t, navAttribute(toggle, "aria-label"), "the glyphs are aria-hidden, so the name lives here")

	// A back-navigation restores form state; without this the menu reopens over the page the
	// visitor just returned to.
	require.Equal(t, "off", navAttribute(toggle, "autocomplete"))

	// hide-visual keeps it FOCUSABLE, which is what makes the label a skin rather than a
	// replacement.  Desktop hides the whole .nav-menu wrapper instead.
	require.True(t, navHasClass(toggle, "hide-visual"))

	label := navFindOne(t, document, "menu label", func(node *html.Node) bool {
		return node.Data == "label" && navAttribute(node, "for") == "nav-menu-toggle"
	})

	require.Empty(t, navAttribute(label, "role"), "role on the label would cost a second tab stop")
	require.Empty(t, navAttribute(label, "tabindex"))
}

// TestMinimalTheme_SheetCopiesAreInvisibleToSelectNav is the cost of rendering the links twice,
// and the two omissions that pay it.  SelectNav resolves the current section with
// getElementById('nav-'+id) -- which returns the FIRST match, not the visible one -- and then
// runs `take .selected from .nav-item`, which would strip the server-rendered mark off whichever
// copy it did not choose.  So the sheet's links carry neither.
func TestMinimalTheme_SheetCopiesAreInvisibleToSelectNav(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainHasRegistrationForm = true

	document := parseMinimalNavigation(t, context)
	panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))

	links := navFind(panel, navIsTag("a"))
	require.NotEmpty(t, links)

	for _, link := range links {
		require.Empty(t, navAttribute(link, "id"),
			"a sheet copy with an id would shadow the bar's copy in getElementById")
		require.False(t, navHasClass(link, "nav-item"),
			"a sheet copy carrying .nav-item would be swept by SelectNav's `take .selected`")
	}

	// The BAR's copies must keep both, or SelectNav stops working altogether
	bar := navFindOne(t, document, "<nav> element", navIsTag("nav"))

	for _, item := range context.Navigation {
		link := navFindOne(t, bar, "bar copy of /"+item.Token, func(node *html.Node) bool {
			return node.Data == "a" && navAttribute(node, "href") == "/"+item.Token
		})

		require.Equal(t, "nav-"+item.StreamID, navAttribute(link, "id"))
		require.True(t, navHasClass(link, "nav-item"))
	}
}

// TestMinimalTheme_EveryLinkRenderedTwice holds the two layouts to the same list.  A link added
// to one and forgotten in the other is invisible on exactly one class of device, which is the
// failure mode this whole design trades duplicate ids to avoid noticing too late.
func TestMinimalTheme_EveryLinkRenderedTwice(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainHasRegistrationForm = true

	document := parseMinimalNavigation(t, context)

	counts := mapof.NewInt()

	for _, link := range navFind(document, navIsTag("a")) {
		counts[navAttribute(link, "href")]++
	}

	for _, href := range []string{"/about", "/contact", "/signin", "/register"} {
		require.Equal(t, 2, counts[href], "%s must render once in the bar and once in the sheet", href)
	}

	require.Equal(t, 1, counts["#main"], "the skip link renders once, outside both layouts")
}

// TestMinimalTheme_SheetMarksCurrentSection covers the server-side replacement for SelectNav,
// including both fallbacks.  .NavigationID is matched against .StreamID while ranging, so an
// id that matches nothing -- and the empty string every Common builder starts with -- simply
// marks nothing rather than marking the first row.
func TestMinimalTheme_SheetMarksCurrentSection(t *testing.T) {

	selectedInSheet := func(t *testing.T, navigationID string) []string {

		t.Helper()

		context := newMinimalNavigationBuilder(t, mapof.NewAny())
		context.NavigationID = navigationID

		document := parseMinimalNavigation(t, context)
		panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))

		var result []string

		// Only the top-level SECTIONS carry a current-section state.  Sign In and Join Now
		// are in the sheet too, and are deliberately not sections, so they carry no
		// aria-current at all -- "false" would claim they are candidates for it.
		for _, item := range context.Navigation {

			link := navFindOne(t, panel, "sheet copy of /"+item.Token, func(node *html.Node) bool {
				return node.Data == "a" && navAttribute(node, "href") == "/"+item.Token
			})

			if navHasClass(link, "selected") {
				require.Equal(t, "page", navAttribute(link, "aria-current"),
					".selected and aria-current must agree")
				result = append(result, navAttribute(link, "href"))
				continue
			}

			// aria-current is always emitted on a section, with the value "false", rather
			// than being conditionally added -- a Go action in TAG position does not survive
			// the template minifier, and "false" is valid ARIA for "not the current item".
			require.Equal(t, "false", navAttribute(link, "aria-current"))
		}

		return result
	}

	require.Equal(t, []string{"/contact"}, selectedInSheet(t, "6a9484676097f8f87ef9a455"))
	require.Equal(t, []string{"/about"}, selectedInSheet(t, "6a908f1d331aab271b96e5f8"))
	require.Empty(t, selectedInSheet(t, ""), "an empty NavigationID must mark nothing")
	require.Empty(t, selectedInSheet(t, "ffffffffffffffffffffffff"), "an unmatched id must mark nothing")
}

// TestMinimalTheme_NavigationToggle_HidesTheMenuToo verifies that showNavigation:false takes the
// mobile menu with it.  The control is fixed to the corner of the SCREEN, so a menu left behind
// by a Domain that switched the navigation off would be the only thing on the page.
func TestMinimalTheme_NavigationToggle_HidesTheMenuToo(t *testing.T) {

	document := parseMinimalNavigation(t, newMinimalNavigationBuilder(t, mapof.Any{"showNavigation": false}))

	require.Empty(t, navFind(document, navIsTag("nav")))
	require.Empty(t, navFind(document, navHasID("nav-menu-toggle")))
	require.Empty(t, navFind(document, navHasID("nav-menu-panel")))
	require.Empty(t, navFind(document, func(node *html.Node) bool {
		return navHasClass(node, "nav-menu")
	}))
}

// TestMinimalTheme_StartupCollapsesLikeAnyOtherState verifies that the STARTUP branch reaches
// the sheet as well as the bar.  It is deliberately NOT special-cased: a branch that exists for
// one screen a server sees once, on the least-exercised path in the theme, is worse than a
// hamburger over a single link.
func TestMinimalTheme_StartupCollapsesLikeAnyOtherState(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainStateID = model.DomainStateStartup

	document := parseMinimalNavigation(t, context)
	panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))

	navFindOne(t, panel, "startup link inside the sheet", func(node *html.Node) bool {
		return node.Data == "a" && navAttribute(node, "href") == "/startup"
	})
}

// TestMinimalTheme_SheetLinksCarryTurboclick pins a deliberate choice that reads like an
// oversight.  turboclick fires navigation on mousedown, and an earlier draft left it off the
// sheet on the theory that a press starting a scroll drag would navigate instead.  It does not:
// on a touch device the browser only synthesises a mousedown once it has already resolved the
// gesture as a tap rather than a scroll.  The bar has always paired turboclick with a scrolling
// container (overflow-x:auto) for the same reason.  Both layouts get it, or the sheet feels
// slower than the bar for no reason anyone can see.
func TestMinimalTheme_SheetLinksCarryTurboclick(t *testing.T) {

	context := newMinimalNavigationBuilder(t, mapof.NewAny())
	context.DomainHasRegistrationForm = true

	document := parseMinimalNavigation(t, context)
	panel := navFindOne(t, document, "menu sheet", navHasID("nav-menu-panel"))

	links := navFind(panel, navIsTag("a"))
	require.NotEmpty(t, links)

	for _, link := range links {
		require.True(t, navHasClass(link, "turboclick"),
			"every sheet link needs turboclick: %s", navAttribute(link, "href"))
	}

	// The STARTUP branch is a separate code path in the template, and it is easy to miss
	startup := newMinimalNavigationBuilder(t, mapof.NewAny())
	startup.DomainStateID = model.DomainStateStartup

	startupPanel := navFindOne(t, parseMinimalNavigation(t, startup), "menu sheet", navHasID("nav-menu-panel"))

	// Scoped to the href: the STARTUP branch replaces only the NAVIGATION links, so an
	// anonymous visitor still gets Sign In beside the checklist.
	startupLink := navFindOne(t, startupPanel, "startup link", func(node *html.Node) bool {
		return node.Data == "a" && navAttribute(node, "href") == "/startup"
	})
	require.True(t, navHasClass(startupLink, "turboclick"))

	// The CONTROL must NOT carry it.  turboclick dispatches a synthetic click on the pressed
	// element; on a <label> bound to the checkbox that toggles the menu twice, landing back
	// where it started -- the same double-toggle that rules popUp out of this design (D3).
	label := navFindOne(t, document, "menu label", func(node *html.Node) bool {
		return node.Data == "label"
	})
	require.False(t, navHasClass(label, "turboclick"))
}
