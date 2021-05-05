package content

import (
	"testing"

	"github.com/benpate/datatype"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func getTestContent() Content {
	return Content{
		{
			Type:  "CONTAINER",
			Refs:  []int{1, 2, 3, 4},
			Check: "home",
			Data: datatype.Map{
				"style": "ROWS",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "secret1",
			Data: datatype.Map{
				"html": "This is the <b>html</b>",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "secret2",
			Data: datatype.Map{
				"html": "This is the second WYSIWYG section",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "secret3",
			Data: datatype.Map{
				"html": "This is the third.",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "secret4",
			Data: datatype.Map{
				"html": "You guessed it.  Fourth section here.",
			},
		},
	}
}

func TestAdd(t *testing.T) {

	content := getTestContent()

	require.NotContains(t, content[0].Refs, 5)

	{
		txn := NewItemTransaction{
			ParentID:   0,
			ChildIndex: 0,
			ItemType:   "WYSIWYG",
			Check:      "home",
		}

		txn.Execute(&content)
	}

	{
		txn := UpdateItemTransaction{
			ItemID: 5,
			Check:  content[5].Check,
			Data: datatype.Map{
				"html": "This is how we do it baby",
			},
		}

		txn.Execute(&content)
	}

	require.Equal(t, 6, len(content))
	require.Equal(t, "WYSIWYG", content[5].Type)
	require.Equal(t, "This is how we do it baby", content[5].Data["html"])
	require.Contains(t, content[0].Refs, 5)
}

func TestCompact(t *testing.T) {

	content := getTestContent()

	content.DeleteReference(0, 3, "home")
	content.Compact()

	spew.Dump(content)
}
