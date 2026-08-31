package service

import (
	"html/template"
	"os"
	"strings"
	"testing"

	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Navigation List Widget
 *
 * These tests execute the real templates in
 * _embed/templates/widget-navigation-list through the real
 * loader, against a stand-in for the build.Widget context.
 * They exist to pin down the tier chain's guards: a tier that
 * cannot render for this Stream must drop out of the chain
 * instead of ending it.
 ******************************************/

// navFakeSummary stands in for model.StreamSummary, which the widget's templates
// address as a Token/Label pair plus a StreamID() accessor.
type navFakeSummary struct {
	Token string
	Label string
	id    string
}

// StreamID returns the hex ID of this summary, matching model.StreamSummary
func (summary navFakeSummary) StreamID() string {
	return summary.id
}

// navFakeQuery stands in for build.QueryBuilder, of which the templates only ever call Slice()
type navFakeQuery struct {
	rows []navFakeSummary
}

// Slice returns every row in this fake query
func (query navFakeQuery) Slice() ([]navFakeSummary, error) {
	return query.rows, nil
}

// navFakeWidget stands in for model.StreamWidget, carrying the widget's saved settings
type navFakeWidget struct {
	Data mapof.Any
}

// navFakeBuilder stands in for build.Widget: the Stream being viewed, plus the
// settings of the navigation-list widget that is rendering on it.
type navFakeBuilder struct {
	Widget      navFakeWidget
	streamID    string
	parentID    string
	token       string
	label       string
	hasParent   bool
	hasGrand    bool
	ancestors   []navFakeSummary
	siblings    []navFakeSummary
	children    []navFakeSummary
	parent      *navFakeBuilder
	grandparent *navFakeBuilder
}

// StreamID returns the hex ID of the Stream being built
func (builder navFakeBuilder) StreamID() string {
	return builder.streamID
}

// ParentID returns the hex ID of this Stream's parent
func (builder navFakeBuilder) ParentID() string {
	return builder.parentID
}

// Token returns the URL token of the Stream being built
func (builder navFakeBuilder) Token() string {
	return builder.token
}

// Label returns the display name of the Stream being built
func (builder navFakeBuilder) Label() string {
	return builder.label
}

// HasParent returns TRUE if this Stream has a parent
func (builder navFakeBuilder) HasParent() bool {
	return builder.hasParent
}

// HasGrandparent returns TRUE if this Stream has a grandparent
func (builder navFakeBuilder) HasGrandparent() bool {
	return builder.hasGrand
}

// Ancestors returns this Stream's parent generation
func (builder navFakeBuilder) Ancestors() navFakeQuery {
	return navFakeQuery{rows: builder.ancestors}
}

// Siblings returns this Stream's own generation
func (builder navFakeBuilder) Siblings() navFakeQuery {
	return navFakeQuery{rows: builder.siblings}
}

// Children returns this Stream's child generation
func (builder navFakeBuilder) Children() navFakeQuery {
	return navFakeQuery{rows: builder.children}
}

// Parent returns a builder for this Stream's parent, or an error if there is none.
// It mirrors build.Stream.Parent, whose error aborts template execution.
func (builder navFakeBuilder) Parent(_ string) (navFakeBuilder, error) {

	if builder.parent == nil {
		return navFakeBuilder{}, derpNavNotFound("parent")
	}

	return *builder.parent, nil
}

// Grandparent returns a builder for this Stream's grandparent, or an error if there is none
func (builder navFakeBuilder) Grandparent(_ string) (navFakeBuilder, error) {

	if builder.grandparent == nil {
		return navFakeBuilder{}, derpNavNotFound("grandparent")
	}

	return *builder.grandparent, nil
}

// derpNavNotFound builds the "no such relative" error that the fake builder returns
func derpNavNotFound(relation string) error {
	return &navMissingRelativeError{relation: relation}
}

// navMissingRelativeError reports that a Stream has no relative of the requested kind
type navMissingRelativeError struct {
	relation string
}

// Error implements the error interface
func (err *navMissingRelativeError) Error() string {
	return "stream has no " + err.relation
}

// renderNavigationWidget parses the real widget-navigation-list folder exactly the way the
// server does at startup, then executes its entry template against the provided context.
func renderNavigationWidget(t *testing.T, context navFakeBuilder) string {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	widgetTemplate := template.New("widget-navigation-list").Funcs(funcMap)

	filesystem := os.DirFS("../_embed/templates/widget-navigation-list")
	require.NoError(t, loadHTMLTemplateFromFilesystem(filesystem, widgetTemplate, funcMap))

	var buffer strings.Builder
	require.NoError(t, widgetTemplate.ExecuteTemplate(&buffer, "widget", context))

	return buffer.String()
}

// newNavTopLevelStream returns a top-level Stream ("Content Pages") that has three children
// and four peers.  This is the shape that exposed the missing .HasParent guard.
func newNavTopLevelStream(settings mapof.Any) navFakeBuilder {

	return navFakeBuilder{
		Widget:    navFakeWidget{Data: settings},
		streamID:  "6a91a6be464adc8963b16d19",
		parentID:  "000000000000000000000000",
		token:     "content",
		label:     "Content Pages",
		hasParent: false,
		hasGrand:  false,

		// A top-level Stream's Ancestors() is empty, because it has no parent generation
		ancestors: []navFakeSummary{},

		siblings: []navFakeSummary{
			{id: "6a908f1d331aab271b96e5f8", Token: "about", Label: "About Me"},
			{id: "6a91a6be464adc8963b16d19", Token: "content", Label: "Content Pages"},
			{id: "6a9484676097f8f87ef9a455", Token: "contact", Label: "Contact Me"},
		},

		children: []navFakeSummary{
			{id: "6a91a715464adc8963b16da5", Token: "wysiwyg", Label: "WYSIWYG"},
			{id: "6a91a715464adc8963b16da6", Token: "markdown", Label: "Markdown"},
		},
	}
}

// TestNavigationWidget_TopLevelStream verifies the reported bug: a top-level Stream configured
// with parents:"All" must still highlight itself, instead of listing its own generation as though
// it were the parent tier and then matching nothing.
func TestNavigationWidget_TopLevelStream(t *testing.T) {

	context := newNavTopLevelStream(mapof.Any{
		"parents":  "All",
		"siblings": "All",
		"children": "All",
	})

	result := renderNavigationWidget(t, context)

	// The current Stream is bold, and is NOT a link
	require.Contains(t, result, `<div class="bold ellipsis">Content Pages</div>`)
	require.NotContains(t, result, `href="/content"`)

	// Its peers are still links
	require.Contains(t, result, `href="/about"`)
	require.Contains(t, result, `href="/contact"`)

	// And the children tier survived the pass-through
	require.Contains(t, result, `href="/wysiwyg"`)
	require.Contains(t, result, `href="/markdown"`)
}

// TestNavigationWidget_TopLevelStreamWithSchemaDefaults verifies that a freshly saved widget,
// which carries "Grandparent Only" and "Parent Only", degrades to the sibling list on a
// top-level Stream rather than aborting on an unguarded .Parent call.
func TestNavigationWidget_TopLevelStreamWithSchemaDefaults(t *testing.T) {

	context := newNavTopLevelStream(mapof.Any{
		"grandparents": "Grandparent Only",
		"parents":      "Parent Only",
		"siblings":     "All",
		"children":     "",
	})

	result := renderNavigationWidget(t, context)

	require.Contains(t, result, `<div class="bold ellipsis">Content Pages</div>`)
	require.Contains(t, result, `href="/about"`)

	// "children" is unset, so no child ever renders
	require.NotContains(t, result, `href="/wysiwyg"`)
}

// TestNavigationWidget_TopLevelStreamWithAllTiers verifies that "All" on every tier is safe on a
// top-level Stream, where neither the grandparent nor the parent tier can produce anything.
func TestNavigationWidget_TopLevelStreamWithAllTiers(t *testing.T) {

	context := newNavTopLevelStream(mapof.Any{
		"grandparents": "All",
		"parents":      "All",
		"siblings":     "All",
		"children":     "All",
	})

	result := renderNavigationWidget(t, context)

	require.Contains(t, result, `<div class="bold ellipsis">Content Pages</div>`)
	require.Contains(t, result, `href="/markdown"`)
}

// TestNavigationWidget_FullHierarchy verifies that a Stream with a full lineage still nests every
// tier under the relative it belongs to.
func TestNavigationWidget_FullHierarchy(t *testing.T) {

	grandparent := navFakeBuilder{
		streamID: "1000",
		parentID: "000000000000000000000000",
		token:    "root",
		label:    "Root",
	}

	parent := navFakeBuilder{
		streamID:  "2000",
		parentID:  "1000",
		token:     "content",
		label:     "Content Pages",
		hasParent: true,

		// The grandparent and the grandparent's peers
		ancestors: []navFakeSummary{
			{id: "1000", Token: "root", Label: "Root"},
			{id: "1001", Token: "other-root", Label: "Other Root"},
		},
	}

	context := navFakeBuilder{
		Widget: navFakeWidget{Data: mapof.Any{
			"grandparents": "All",
			"parents":      "All",
			"siblings":     "All",
			"children":     "All",
		}},
		streamID:  "3000",
		parentID:  "2000",
		token:     "markdown",
		label:     "Markdown",
		hasParent: true,
		hasGrand:  true,

		ancestors: []navFakeSummary{
			{id: "2000", Token: "content", Label: "Content Pages"},
			{id: "2001", Token: "photos", Label: "Photos"},
		},

		siblings: []navFakeSummary{
			{id: "3000", Token: "markdown", Label: "Markdown"},
			{id: "3001", Token: "wysiwyg", Label: "WYSIWYG"},
		},

		children: []navFakeSummary{
			{id: "4000", Token: "example", Label: "Example"},
		},

		parent:      &parent,
		grandparent: &grandparent,
	}

	result := renderNavigationWidget(t, context)

	// Every tier rendered
	require.Contains(t, result, `href="/root"`)
	require.Contains(t, result, `href="/other-root"`)
	require.Contains(t, result, `href="/content"`)
	require.Contains(t, result, `href="/photos"`)
	require.Contains(t, result, `href="/wysiwyg"`)
	require.Contains(t, result, `href="/example"`)

	// The current Stream is bold, and is NOT a link
	require.Contains(t, result, `<div class="bold ellipsis">Markdown</div>`)
	require.NotContains(t, result, `href="/markdown"`)

	// Each tier nests below the relative it belongs to, and each nested tier interrupts its
	// generation at the matching entry rather than following it:
	//
	//   Root
	//     Content Pages
	//       Markdown  (current)
	//         Example
	//       WYSIWYG
	//     Photos
	//   Other Root
	order := []string{
		`href="/root"`,
		`href="/content"`,
		`<div class="bold ellipsis">Markdown</div>`,
		`href="/example"`,
		`href="/wysiwyg"`,
		`href="/photos"`,
		`href="/other-root"`,
	}

	for index := 1; index < len(order); index++ {
		require.Less(t, strings.Index(result, order[index-1]), strings.Index(result, order[index]),
			"%s must precede %s", order[index-1], order[index])
	}
}

// TestNavigationWidget_ParentMissingFromAncestors verifies that a parent which is filtered out of
// the ancestor list by permissions or publish date does not swallow the tiers below it.
func TestNavigationWidget_ParentMissingFromAncestors(t *testing.T) {

	context := navFakeBuilder{
		Widget: navFakeWidget{Data: mapof.Any{
			"parents":  "All",
			"siblings": "All",
			"children": "All",
		}},
		streamID:  "3000",
		parentID:  "2000",
		token:     "markdown",
		label:     "Markdown",
		hasParent: true,

		// The real parent ("2000") is absent from its own generation
		ancestors: []navFakeSummary{
			{id: "2001", Token: "photos", Label: "Photos"},
		},

		siblings: []navFakeSummary{
			{id: "3000", Token: "markdown", Label: "Markdown"},
			{id: "3001", Token: "wysiwyg", Label: "WYSIWYG"},
		},

		children: []navFakeSummary{
			{id: "4000", Token: "example", Label: "Example"},
		},
	}

	result := renderNavigationWidget(t, context)

	require.Contains(t, result, `<div class="bold ellipsis">Markdown</div>`)
	require.Contains(t, result, `href="/wysiwyg"`)
	require.Contains(t, result, `href="/example"`)
}

// TestNavigationWidget_StreamMissingFromSiblings verifies that an unpublished Stream, which its own
// sibling query filters out, still renders its children.
func TestNavigationWidget_StreamMissingFromSiblings(t *testing.T) {

	context := navFakeBuilder{
		Widget: navFakeWidget{Data: mapof.Any{
			"siblings": "All",
			"children": "All",
		}},
		streamID: "3000",
		token:    "markdown",
		label:    "Markdown",

		siblings: []navFakeSummary{
			{id: "3001", Token: "wysiwyg", Label: "WYSIWYG"},
		},

		children: []navFakeSummary{
			{id: "4000", Token: "example", Label: "Example"},
		},
	}

	result := renderNavigationWidget(t, context)

	require.Contains(t, result, `href="/wysiwyg"`)
	require.Contains(t, result, `href="/example"`)
}

// TestNavigationWidget_NoTiersConfigured verifies that a widget with no tiers set renders its
// title and nothing else, instead of failing somewhere along the chain.
func TestNavigationWidget_NoTiersConfigured(t *testing.T) {

	context := newNavTopLevelStream(mapof.Any{
		"title": "Table of Contents",
	})

	result := renderNavigationWidget(t, context)

	require.Contains(t, result, `Table of Contents`)
	require.NotContains(t, result, `ellipsis`)
}
