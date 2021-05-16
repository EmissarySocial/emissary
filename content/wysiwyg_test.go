package content

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWYSIWYG(t *testing.T) {

	var c Content
	s := []byte(`[{
		"type":"WYSIWYG",
		"check": "123456789101112",
		"data":{
			"html":"This is some <i>HTML</i>"
		}}]`)

	err := json.Unmarshal(s, &c)

	require.Nil(t, err)

	{
		html := c.View()
		require.Equal(t, "This is some <i>HTML</i>", html)
	}

	{
		html := c.Edit("/my-url")
		expected := `<form method="post" action="/my-url" data-script="install wysiwyg"><input type="hidden" name="type" value="update-item"><input type="hidden" name="itemId" value="0"><input type="hidden" name="check" value="123456789101112"><input type="hidden" name="html"><div class="ck-editor">This is some <i>HTML</i></div></form>`
		require.Equal(t, expected, html)
	}
}
