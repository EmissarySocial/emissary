package transaction

import (
	"testing"

	"github.com/benpate/datatype"
	"github.com/benpate/ghost/content"
	"github.com/stretchr/testify/require"
)

func TestAddItem_AppendContainer_Above(t *testing.T) {

	c := content.Content{
		{
			Type:  "CONTAINER",
			Check: "123",
			Refs:  []int{1},
			Data: datatype.Map{
				"style": "ROWS",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "123",
			Data: datatype.Map{
				"html": "This is the first item",
			},
		},
	}
	txn := NewItem{
		ItemID: 0,
		Place:  "ABOVE",
		Type:   "HTML",
		Check:  "123",
	}

	err := txn.Execute(&c)

	require.Nil(t, err)
	require.Equal(t, "CONTAINER", c[0].Type)
	require.Equal(t, "ROWS", c[0].Data["style"])
	require.Equal(t, []int{2, 1}, c[0].Refs)

	require.Equal(t, "WYSIWYG", c[1].Type)
	require.Equal(t, "This is the first item", c[1].Data["html"])

	require.Equal(t, "HTML", c[2].Type)
	require.Empty(t, c[2].Data["html"])

	expected := `<div class="container" data-style="ROWS" data-size="2"><div class="container-item"></div><div class="container-item">This is the first item</div></div>`
	require.Equal(t, expected, c.View())
}

func TestAddItem_AppendContainer_Below(t *testing.T) {

	c := content.Content{
		{
			Type:  "CONTAINER",
			Check: "123",
			Refs:  []int{1},
			Data: datatype.Map{
				"style": "ROWS",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "123",
			Data: datatype.Map{
				"html": "This is the first item",
			},
		},
	}
	txn := NewItem{
		ItemID: 0,
		Place:  "BELOW",
		Type:   "HTML",
		Check:  "123",
	}

	err := txn.Execute(&c)

	require.Nil(t, err)
	require.Equal(t, "CONTAINER", c[0].Type)
	require.Equal(t, "ROWS", c[0].Data["style"])
	require.Equal(t, []int{1, 2}, c[0].Refs)

	require.Equal(t, "WYSIWYG", c[1].Type)
	require.Equal(t, "This is the first item", c[1].Data["html"])

	require.Equal(t, "HTML", c[2].Type)
	require.Empty(t, c[2].Data["html"])

	expected := `<div class="container" data-style="ROWS" data-size="2"><div class="container-item">This is the first item</div><div class="container-item"></div></div>`
	require.Equal(t, expected, c.View())
}

func TestAddItem_AppendContainer_Left(t *testing.T) {

	c := content.Content{
		{
			Type:  "CONTAINER",
			Check: "123",
			Refs:  []int{1},
			Data: datatype.Map{
				"style": "COLS",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "123",
			Data: datatype.Map{
				"html": "This is the first item",
			},
		},
	}
	txn := NewItem{
		ItemID: 0,
		Place:  "LEFT",
		Type:   "HTML",
		Check:  "123",
	}

	err := txn.Execute(&c)

	require.Nil(t, err)
	require.Equal(t, "CONTAINER", c[0].Type)
	require.Equal(t, "COLS", c[0].Data["style"])
	require.Equal(t, []int{2, 1}, c[0].Refs)

	require.Equal(t, "WYSIWYG", c[1].Type)
	require.Equal(t, "This is the first item", c[1].Data["html"])

	require.Equal(t, "HTML", c[2].Type)
	require.Empty(t, c[2].Data["html"])

	expected := `<div class="container" data-style="COLS" data-size="2"><div class="container-item"></div><div class="container-item">This is the first item</div></div>`
	require.Equal(t, expected, c.View())
}

func TestAddItem_AppendContainer_Right(t *testing.T) {

	c := content.Content{
		{
			Type:  "CONTAINER",
			Check: "123",
			Refs:  []int{1},
			Data: datatype.Map{
				"style": "COLS",
			},
		},
		{
			Type:  "WYSIWYG",
			Check: "123",
			Data: datatype.Map{
				"html": "This is the first item",
			},
		},
	}
	txn := NewItem{
		ItemID: 0,
		Place:  "RIGHT",
		Type:   "HTML",
		Check:  "123",
	}

	err := txn.Execute(&c)

	require.Nil(t, err)
	require.Equal(t, "CONTAINER", c[0].Type)
	require.Equal(t, "COLS", c[0].Data["style"])
	require.Equal(t, []int{1, 2}, c[0].Refs)

	require.Equal(t, "WYSIWYG", c[1].Type)
	require.Equal(t, "This is the first item", c[1].Data["html"])

	require.Equal(t, "HTML", c[2].Type)
	require.Empty(t, c[2].Data["html"])

	expected := `<div class="container" data-style="COLS" data-size="2"><div class="container-item">This is the first item</div><div class="container-item"></div></div>`
	require.Equal(t, expected, c.View())
}
