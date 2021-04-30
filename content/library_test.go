package content

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLibrary(t *testing.T) {
	lib := ViewerLibrary()
	item := getTestItem()
	result := lib.Render(&item)
	require.Equal(t, "<b>This</b> is a test object", result)
}
