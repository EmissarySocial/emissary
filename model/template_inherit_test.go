package model

import (
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
