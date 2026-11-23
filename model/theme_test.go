package model

import (
	"testing"

	"github.com/benpate/form"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestTheme_Inherit_StartupTasks verifies that a Theme with no startup tasks takes its parent's list
func TestTheme_Inherit_StartupTasks(t *testing.T) {

	parent := NewTheme("parent", nil)
	parent.StartupTasks = append(parent.StartupTasks, form.LookupCode{Value: "/admin/domain", Label: "Website Name"})

	child := NewTheme("child", nil)
	child.Inherit(&parent)

	require.Len(t, child.StartupTasks, 1)
	require.Equal(t, "/admin/domain", child.StartupTasks[0].Value)
	require.Equal(t, "Website Name", child.StartupTasks[0].Label)
}

// TestTheme_Inherit_KeepsOwnStartupTasks verifies that a Theme's own startup tasks replace its parent's list whole
func TestTheme_Inherit_KeepsOwnStartupTasks(t *testing.T) {

	parent := NewTheme("parent", nil)
	parent.StartupTasks = append(parent.StartupTasks, form.LookupCode{Value: "/admin/domain"}, form.LookupCode{Value: "/admin/rules"})

	child := NewTheme("child", nil)
	child.StartupTasks = append(child.StartupTasks, form.LookupCode{Value: "/admin/searchTags"})
	child.Inherit(&parent)

	require.Len(t, child.StartupTasks, 1)
	require.Equal(t, "/admin/searchTags", child.StartupTasks[0].Value)
}

// TestTheme_Inherit_StartupDefaultPages verifies that a Theme naming no landing pages takes its parent's
func TestTheme_Inherit_StartupDefaultPages(t *testing.T) {

	parent := NewTheme("parent", nil)
	parent.StartupDefaultPages = StartupDefaultPages{Anonymous: "/home", Authenticated: "/@me/newsfeed", Owner: "/admin"}

	child := NewTheme("child", nil)
	child.Inherit(&parent)

	require.Equal(t, parent.StartupDefaultPages, child.StartupDefaultPages)
}

// TestTheme_Inherit_KeepsOwnStartupDefaultPages verifies that a Theme's own landing pages are not mixed with its parent's
func TestTheme_Inherit_KeepsOwnStartupDefaultPages(t *testing.T) {

	parent := NewTheme("parent", nil)
	parent.StartupDefaultPages = StartupDefaultPages{Anonymous: "/home", Authenticated: "/@me/newsfeed", Owner: "/admin"}

	child := NewTheme("child", nil)
	child.StartupDefaultPages = StartupDefaultPages{Authenticated: "/@me"}
	child.Inherit(&parent)

	require.Equal(t, StartupDefaultPages{Authenticated: "/@me"}, child.StartupDefaultPages)
}

// TestTheme_Inherit_NilParent verifies that a missing parent changes nothing
func TestTheme_Inherit_NilParent(t *testing.T) {

	child := NewTheme("child", nil)
	child.StartupDefaultPages = StartupDefaultPages{Anonymous: "/home"}
	child.Inherit(nil)

	require.Equal(t, StartupDefaultPages{Anonymous: "/home"}, child.StartupDefaultPages)
	require.Empty(t, child.StartupTasks)
}

// TestTheme_StartupDefaultPages_ParsesFromDefinition verifies the spelling that a theme.hjson file uses
func TestTheme_StartupDefaultPages_ParsesFromDefinition(t *testing.T) {

	definition := `{
		themeId: "example"
		startupDefaultPages: {
			anonymous: "/home"
			authenticated: "/@me/newsfeed"
			owner: "/admin"
		}
	}`

	theme := NewTheme("example", nil)
	require.NoError(t, hjson.Unmarshal([]byte(definition), &theme))

	require.Equal(t, StartupDefaultPages{Anonymous: "/home", Authenticated: "/@me/newsfeed", Owner: "/admin"}, theme.StartupDefaultPages)
}

// TestTheme_PropertyForm verifies that only the startup checklist draws a Theme's startup form
func TestTheme_PropertyForm(t *testing.T) {

	theme := NewTheme("example", nil)
	theme.Form = form.Element{Type: "layout-vertical"}
	theme.StartupForm = form.Element{Type: "layout-tabs"}

	require.Equal(t, "layout-tabs", theme.PropertyForm(ThemeStartupTemplateID).Type)
	require.Equal(t, "layout-vertical", theme.PropertyForm("admin-domain").Type)
	require.Equal(t, "layout-vertical", theme.PropertyForm("").Type)
}

// TestTheme_PropertyForm_NoStartupForm verifies that the startup checklist falls back to the Theme's own form
func TestTheme_PropertyForm_NoStartupForm(t *testing.T) {

	theme := NewTheme("example", nil)
	theme.Form = form.Element{Type: "layout-vertical"}

	require.Equal(t, "layout-vertical", theme.PropertyForm(ThemeStartupTemplateID).Type)
}

// TestTheme_StartupForm_ParsesFromDefinition verifies the spelling that a theme.hjson file uses
func TestTheme_StartupForm_ParsesFromDefinition(t *testing.T) {

	definition := `{
		themeId: "example"
		startupForm: {type: "layout-tabs", children: [{type: "layout-vertical", label: "General"}]}
	}`

	theme := NewTheme("example", nil)
	require.NoError(t, hjson.Unmarshal([]byte(definition), &theme))

	require.Equal(t, "layout-tabs", theme.StartupForm.Type)
	require.Len(t, theme.StartupForm.Children, 1)
	require.Equal(t, "General", theme.StartupForm.Children[0].Label)
}
