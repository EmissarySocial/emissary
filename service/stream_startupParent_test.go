package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// newStartupParentService returns a Stream service that knows a "search" page, which may sit at the
// top level or under another search page, and a "top-only" page
func newStartupParentService() *Stream {

	search := model.NewTemplate("search-page", nil)
	search.TemplateRole = "search"
	search.ContainedBy = []string{"top", "search"}

	topOnly := model.NewTemplate("top-only", nil)
	topOnly.TemplateRole = "article"
	topOnly.ContainedBy = []string{"top"}

	return &Stream{templateService: &Template{templates: set.Map[model.Template]{
		"search-page": search,
		"top-only":    topOnly,
	}}}
}

// newStartupParent returns a saved top-level startup Stream, and the map that Startup keeps it in
func newStartupParent() (model.Stream, map[string]model.Stream) {

	parent := model.NewStream()
	parent.Token.Set("albums")
	parent.TemplateID = "search-page"

	return parent, map[string]model.Stream{"albums": parent}
}

// TestStream_setStartupParent_PlacesChild verifies that a child takes its parent's ID, breadcrumbs, and navigation
func TestStream_setStartupParent_PlacesChild(t *testing.T) {

	streamService := newStartupParentService()
	parent, created := newStartupParent()

	child := model.NewStream()
	child.Token.Set("artists")
	child.TemplateID = "search-page"

	require.NoError(t, streamService.setStartupParent(&child, "albums", created))

	require.Equal(t, parent.StreamID, child.ParentID)
	require.Len(t, child.ParentIDs, 1)
	require.Equal(t, parent.StreamID, child.ParentIDs[0])
	require.Equal(t, "search-page", child.ParentTemplateID)
	require.Equal(t, parent.StreamID.Hex(), child.NavigationID, "the child highlights its parent in the navigation")

	// The parent that Startup remembers is not changed by placing a child under it
	require.Empty(t, created["albums"].NavigationID)
	require.Empty(t, created["albums"].ParentIDs)
}

// TestStream_setStartupParent_KeepsParentNavigation verifies that a parent's own NavigationID wins over its ID
func TestStream_setStartupParent_KeepsParentNavigation(t *testing.T) {

	streamService := newStartupParentService()
	parent, created := newStartupParent()
	parent.NavigationID = "elsewhere"
	created["albums"] = parent

	child := model.NewStream()
	child.TemplateID = "search-page"

	require.NoError(t, streamService.setStartupParent(&child, "albums", created))
	require.Equal(t, "elsewhere", child.NavigationID)
}

// TestStream_setStartupParent_NoParent verifies that an entry without a parent stays at the top level
func TestStream_setStartupParent_NoParent(t *testing.T) {

	streamService := newStartupParentService()
	_, created := newStartupParent()

	stream := model.NewStream()
	stream.TemplateID = "search-page"

	require.NoError(t, streamService.setStartupParent(&stream, "", created))
	require.True(t, stream.ParentID.IsZero())
	require.Empty(t, stream.ParentIDs)
}

// TestStream_setStartupParent_Refusals covers each entry that cannot be placed
func TestStream_setStartupParent_Refusals(t *testing.T) {

	streamService := newStartupParentService()

	check := func(templateID string, parentToken string, created map[string]model.Stream, message string) {
		t.Helper()

		stream := model.NewStream()
		stream.TemplateID = templateID

		require.Error(t, streamService.setStartupParent(&stream, parentToken, created), message)
		require.True(t, stream.ParentID.IsZero(), message)
	}

	_, created := newStartupParent()

	check("search-page", "missing", created, "a parent that is not listed")
	check("search-page", "albums", nil, "a parent that is listed later, so nothing is created yet")
	check("top-only", "albums", created, "a template that its parent cannot contain")
	check("unknown", "albums", created, "a template that does not load")
}

// TestStream_newStartupStream_ExcludesParentToken verifies that "parentToken" is not treated as a Stream field
func TestStream_newStartupStream_ExcludesParentToken(t *testing.T) {

	streamService := &Stream{}

	data := mapof.Any{"templateId": "search-page", "token": "artists", "parentToken": "albums"}

	stream, err := streamService.newStartupStream(data)
	require.NoError(t, err)
	require.Equal(t, "artists", stream.Token.Value())
	require.True(t, stream.ParentID.IsZero(), "the builder places nothing; Startup does")
	require.Contains(t, data, "parentToken", "the shared Theme map is not changed")
}
