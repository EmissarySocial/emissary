package content

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLibrary(t *testing.T) {
	lib := ViewerLibrary()
	content := getTestContent()
	result := content.Render(lib)
	expected := `<div class="container" data-style="ROWS" data-size="4"><div class="container-item">This is the <b>html</b></div><div class="container-item">This is the second WYSIWYG section</div><div class="container-item">This is the third.</div><div class="container-item">You guessed it.  Fourth section here.</div></div>`
	require.Equal(t, expected, result)
}
