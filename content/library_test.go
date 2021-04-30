package content

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLibrary(t *testing.T) {
	lib := ViewerLibrary()
	content := getTestContent()
	result := content.Render(lib)
	expected := `<div class="container container-ROWS container-size-4"><div>This is the <b>html</b></div><div>This is the second WYSIWYG section</div><div>This is the third.</div><div>You guessed it.  Fourth section here.</div></div>`
	require.Equal(t, expected, result)
}
