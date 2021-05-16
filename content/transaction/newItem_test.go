package transaction

import (
	"testing"

	"github.com/benpate/datatype"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	content := getTestContent()

	require.NotContains(t, content[0].Refs, 5)

	{
		txn := NewItem{
			ParentID:   0,
			ChildIndex: 0,
			ItemType:   "WYSIWYG",
			Check:      "home",
		}

		txn.Execute(&content)
	}

	{
		txn := UpdateItem{
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
