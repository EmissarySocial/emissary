package model

import (
	"strings"
	"testing"

	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestTemplate_ParseTagFields confirms the hjson keys "tagPaths" and "tagUrl" map to the Template fields.
func TestTemplate_ParseTagFields(t *testing.T) {
	template := NewTemplate("test", nil)

	err := hjson.Unmarshal([]byte(`{tagPaths:["content.html"], tagUrl:"/search?q="}`), &template)

	require.Nil(t, err)
	require.Equal(t, []string{"content.html"}, template.TagPaths)
	require.Equal(t, "/search?q=", template.TagURL)
}

// TestTemplate_Inherit_TagURL confirms that an empty child inherits the parent's TagURL.
func TestTemplate_Inherit_TagURL(t *testing.T) {
	parent := NewTemplate("parent", nil)
	parent.TagURL = "/search?q="

	child := NewTemplate("child", nil)
	child.Inherit(&parent)

	require.Equal(t, "/search?q=", child.TagURL)
}

// TestTemplate_Inherit_TagURL_ChildOverride confirms that a child's own TagURL is not overwritten.
func TestTemplate_Inherit_TagURL_ChildOverride(t *testing.T) {
	parent := NewTemplate("parent", nil)
	parent.TagURL = "/search?q="

	child := NewTemplate("child", nil)
	child.TagURL = "/home?q="
	child.Inherit(&parent)

	require.Equal(t, "/home?q=", child.TagURL)
}

// TestTemplate_Inherit_TagPathsNotInherited confirms that TagPaths deliberately does NOT inherit.
// External template packages depend on this: each concrete template must declare its own tagPaths.
func TestTemplate_Inherit_TagPathsNotInherited(t *testing.T) {
	parent := NewTemplate("parent", nil)
	parent.TagPaths = []string{"data.tags"}

	child := NewTemplate("child", nil)
	child.Inherit(&parent)

	require.Empty(t, child.TagPaths)
}

// TestTemplate_Inherit_HTMLTemplate confirms a child receives the parent's named templates.
// This is what lets base-widget-editor ship a shared canvas.
func TestTemplate_Inherit_HTMLTemplate(t *testing.T) {

	parent := NewTemplate("parent", nil)
	_, err := parent.HTMLTemplate.New("layout-controls").Parse(`FROM-PARENT`)
	require.Nil(t, err)

	child := NewTemplate("child", nil)
	child.Inherit(&parent)

	require.NotNil(t, child.HTMLTemplate.Lookup("layout-controls"))
	require.Equal(t, "FROM-PARENT", executeNamed(t, &child, "layout-controls"))
}

// TestTemplate_Inherit_HTMLTemplate_ChildWins confirms that a child's own definition of a
// named template is NOT replaced by the parent's.  The "layout-controls" slot depends on
// this: base-widget-editor ships an empty default, and any template that wants controls
// simply defines its own and has it win.
func TestTemplate_Inherit_HTMLTemplate_ChildWins(t *testing.T) {

	parent := NewTemplate("parent", nil)
	_, err := parent.HTMLTemplate.New("layout-controls").Parse(`FROM-PARENT`)
	require.Nil(t, err)

	child := NewTemplate("child", nil)
	_, err = child.HTMLTemplate.New("layout-controls").Parse(`FROM-CHILD`)
	require.Nil(t, err)

	child.Inherit(&parent)

	require.Equal(t, "FROM-CHILD", executeNamed(t, &child, "layout-controls"))
}

// executeNamed renders one named template from a Template's parse tree
func executeNamed(t *testing.T, template *Template, name string) string {
	t.Helper()
	var buffer strings.Builder
	require.Nil(t, template.HTMLTemplate.ExecuteTemplate(&buffer, name, nil))
	return buffer.String()
}
