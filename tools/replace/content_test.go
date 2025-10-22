package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContent_Naive(t *testing.T) {
	original := "This is an example of a string to replace"
	result := Content(original, "is", "was")
	require.Equal(t, "Thwas was an example of a string to replace", result)
}

func TestContent_AtBeginning(t *testing.T) {
	original := "#Here's a tag at the beginning"
	result := Content(original, "#Here", "<b>#Here</b>")
	require.Equal(t, "<b>#Here</b>'s a tag at the beginning", result)
}

func TestContent_AtEnd(t *testing.T) {
	original := "Here's a tag at the #end"
	result := Content(original, "#end", "<b>#end</b>")
	require.Equal(t, "Here's a tag at the <b>#end</b>", result)
}

func TestContent_SkipHTML(t *testing.T) {
	original := "Here's some <a href='server.com/#tag'>HTML</a> with a #tag somewhere"
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, "Here's some <a href='server.com/#tag'>HTML</a> with a <b>#tag</b> somewhere", result)
}

func TestContent_SkipHTML_AtEnd(t *testing.T) {
	original := "Here's some <a href='server.com/#tag'>HTML</a> with a #tag at the #end"
	result := Content(original, "#end", "<b>#end</b>")
	require.Equal(t, "Here's some <a href='server.com/#tag'>HTML</a> with a #tag at the <b>#end</b>", result)
}

func TestContent_CaseInsensitive(t *testing.T) {
	original := "THIS this ThIs tHiS is case insensitive"
	result := Content(original, "this", "THAT")
	require.Equal(t, "THAT THAT THAT THAT is case insensitive", result)
}
