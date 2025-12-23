package handler

import (
	"regexp"
	"testing"

	"github.com/benpate/rosetta/convert"
	"github.com/stretchr/testify/require"
)

func TestOEmbed(t *testing.T) {

	var height int
	var width int

	html := `<html><div style="max-height:100px; max-width:200px;">Here's some stuff</div></html>`
	findWidth := regexp.MustCompile(`max-width:\s*(\d+)px;`)
	findHeight := regexp.MustCompile(`max-height:\s*(\d+)px;`)

	if heightStrings := findHeight.FindStringSubmatch(html); len(heightStrings) == 2 {
		height = convert.Int(heightStrings[1])
	}

	require.Equal(t, 100, height)

	if widthStrings := findWidth.FindStringSubmatch(html); len(widthStrings) == 2 {
		width = convert.Int(widthStrings[1])
	}

	require.Equal(t, 200, width)
}
