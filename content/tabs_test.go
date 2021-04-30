package content

import (
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestTabs(t *testing.T) {

	text := []byte(`{
		"type":"TABS",
		"map": {
			"labels": ["First Tab", "Second Tab", "Third Tab"]
		},
		"children": [{
			"type":"HTML",
			"map": {
				"html":"This is the HTML for the first tab."
			}
		},{
			"type":"TEXT",
			"map": {
				"text":"This is the text for the second tab."
			}
		},{
			"type":"HTML",
			"map": {
				"html":"This is the text for the third tab."
			}
		}]
	}`)

	var item Item

	err := json.Unmarshal(text, &item)
	require.Nil(t, err)

	lib := ViewerLibrary()

	result := lib.Render(&item)

	spew.Dump(result)
}
