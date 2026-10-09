package service

import (
	"html/template"
	"io"
	"testing"
	"testing/fstest"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestTheme_Reload_ExecutedMidReload confirms a request that renders a theme partway
// through a reload cannot stop that theme from inheriting its parent's templates
func TestTheme_Reload_ExecutedMidReload(t *testing.T) {

	service := newReloadTestThemes(t)

	// A request renders the child before inheritance runs.  Whichever copy it
	// gets, the render must not mark the copy being built as executed
	requestTheme := service.GetTheme("child")
	_ = requestTheme.HTMLTemplate.ExecuteTemplate(io.Discard, "page", "request")

	// Finish the reload
	service.calculateAllInheritance()
	service.publish()

	// The published child carries the inherited template, and renders it
	published := service.GetTheme("child")
	require.NotNil(t, published.HTMLTemplate.Lookup("user-signin"), "child theme must inherit user-signin from global")
	require.NoError(t, published.HTMLTemplate.ExecuteTemplate(io.Discard, "user-signin", "request"))
}

// TestTheme_Reload_UnpublishedUntilComplete confirms a reload's themes stay out of the
// live library until inheritance has run and they are published
func TestTheme_Reload_UnpublishedUntilComplete(t *testing.T) {

	service := newReloadTestThemes(t)

	// Before publishing, a request cannot see the child
	require.Empty(t, service.List())
	require.Nil(t, service.GetTheme("child").HTMLTemplate.Lookup("page"))

	// After publishing, it can
	service.calculateAllInheritance()
	service.publish()

	require.NotNil(t, service.GetTheme("child").HTMLTemplate.Lookup("page"))
}

// TestTheme_Reload_UnknownParent confirms a theme that extends a theme no location defines
// is reported, and still inherits from the parents that do exist
func TestTheme_Reload_UnknownParent(t *testing.T) {

	service := newReloadTestThemes(t)

	orphan := fstest.MapFS{"page.html": {Data: []byte(`<main>{{.}}</main>`)}}
	require.NoError(t, service.Add("orphan", orphan, []byte(`{extends:["missing", "global"]}`)))

	errs := service.unknownParents()
	require.Len(t, errs, 1)
	require.Contains(t, derp.Details(errs[0]), "parentId: missing")

	theme := service.calculateInheritance(service.themePrep["orphan"])
	require.NotNil(t, theme.HTMLTemplate.Lookup("user-signin"), "known parents must still be inherited")
}

// newReloadTestThemes returns a Theme service holding a "global" theme and a "child"
// that extends it, both added but not yet inherited, as a reload leaves them
func newReloadTestThemes(t *testing.T) *Theme {

	t.Helper()

	service := NewTheme(nil, nil, template.FuncMap{})

	global := fstest.MapFS{"user-signin.html": {Data: []byte(`<p>{{.}}</p>`)}}
	require.NoError(t, service.Add("global", global, []byte(`{}`)))

	child := fstest.MapFS{"page.html": {Data: []byte(`<main>{{.}}</main>`)}}
	require.NoError(t, service.Add("child", child, []byte(`{extends:["global"]}`)))

	return &service
}
