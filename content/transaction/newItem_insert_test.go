package transaction

import (
	"testing"

	"github.com/benpate/datatype"
	"github.com/benpate/ghost/content"
	"github.com/stretchr/testify/require"
)

func TestAddItem_InsertContainer_Above(t *testing.T) {

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
		ItemID:   1,
		Place:    "ABOVE",
		ItemType: "HTML",
		Check:    "123",
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

func TestAddItem_InsertContainer_Above2(t *testing.T) {

	c := content.Content{
		{
			Type:  "CONTAINER",
			Check: "123",
			Refs:  []int{1, 2},
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
		{
			Type:  "WYSIWYG",
			Check: "123",
			Data: datatype.Map{
				"html": "This is the second item",
			},
		},
	}
	txn := NewItem{
		ItemID:   2,
		Place:    "ABOVE",
		ItemType: "HTML",
		Check:    "123",
	}

	err := txn.Execute(&c)

	require.Nil(t, err)
	require.Equal(t, "CONTAINER", c[0].Type)
	require.Equal(t, "ROWS", c[0].Data["style"])
	require.Equal(t, []int{1, 3, 2}, c[0].Refs)

	require.Equal(t, "WYSIWYG", c[1].Type)
	require.Equal(t, "This is the first item", c[1].Data["html"])

	require.Equal(t, "WYSIWYG", c[2].Type)
	require.Equal(t, "This is the second item", c[2].Data["html"])

	require.Equal(t, "HTML", c[3].Type)
	require.Empty(t, c[3].Data["html"])

	expected := `<div class="container" data-style="ROWS" data-size="3"><div class="container-item">This is the first item</div><div class="container-item"></div><div class="container-item">This is the second item</div></div>`
	require.Equal(t, expected, c.View())
}

func TestAddItem_InsertContainer_Below(t *testing.T) {

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
		ItemID:   1,
		Place:    "BELOW",
		ItemType: "HTML",
		Check:    "123",
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

func TestAddItem_InsertContainer_Left(t *testing.T) {

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
		ItemID:   1,
		Place:    "LEFT",
		ItemType: "HTML",
		Check:    "123",
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

func TestAddItem_InsertContainer_Right(t *testing.T) {

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
		ItemID:   1,
		Place:    "RIGHT",
		ItemType: "HTML",
		Check:    "123",
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
