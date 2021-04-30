package content

import (
	"encoding/json"
	"testing"

	"github.com/benpate/datatype"
	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	item := getTestItem()

	require.Equal(t, "HTML", item.Type)
	require.Equal(t, "<b>This</b> is a test object", item.GetString("html"))
}

func TestItemDecode(t *testing.T) {

	var item Item
	encoded := `{"type":"TEXT", "data":{"text": "Hello There"}}`

	err := json.Unmarshal([]byte(encoded), &item)

	require.Nil(t, err)
	require.Equal(t, "TEXT", item.Type)
	require.Equal(t, "Hello There", item.GetString("text"))
}

func getTestItem() Item {

	return Item{
		Type: ItemTypeHTML,
		Data: datatype.Map{
			"html": "<b>This</b> is a test object",
		},
	}
}
