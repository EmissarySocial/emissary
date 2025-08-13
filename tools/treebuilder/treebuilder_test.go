package treebuilder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {

	original := []Item{
		{ItemID: "1", ParentID: "", Name: "One"},
		{ItemID: "2", ParentID: "1", Name: "Two"},
		{ItemID: "3", ParentID: "1", Name: "Three"},
		{ItemID: "4", ParentID: "2", Name: "Four"},
		{ItemID: "5", ParentID: "2", Name: "Five"},
		{ItemID: "6", ParentID: "3", Name: "Six"},
		{ItemID: "7", ParentID: "3", Name: "Seven"},
		{ItemID: "8", ParentID: "4", Name: "Eight"},
		{ItemID: "9", ParentID: "4", Name: "Nine"},
		{ItemID: "10", ParentID: "5", Name: "Ten"},
		{ItemID: "11", ParentID: "5", Name: "Eleven"},
		{ItemID: "12", ParentID: "6", Name: "Twelve"},
		{ItemID: "13", ParentID: "6", Name: "Thirteen"},
		{ItemID: "14", ParentID: "7", Name: "Fourteen"},
		{ItemID: "15", ParentID: "7", Name: "Fifteen"},
	}

	mapped := Parse(original)

	require.Equal(t, mapped.ItemID, "1")

	require.Equal(t, mapped.Children[0].ItemID, "2")
	require.Equal(t, mapped.Children[1].ItemID, "3")

	require.Equal(t, mapped.Children[0].Children[0].ItemID, "4")
	require.Equal(t, mapped.Children[0].Children[1].ItemID, "5")
	require.Equal(t, mapped.Children[1].Children[0].ItemID, "6")
	require.Equal(t, mapped.Children[1].Children[1].ItemID, "7")

	require.Equal(t, mapped.Children[0].Children[0].Children[0].ItemID, "8")
	require.Equal(t, mapped.Children[0].Children[0].Children[1].ItemID, "9")
	require.Equal(t, mapped.Children[0].Children[1].Children[0].ItemID, "10")
	require.Equal(t, mapped.Children[0].Children[1].Children[1].ItemID, "11")
	require.Equal(t, mapped.Children[1].Children[0].Children[0].ItemID, "12")
	require.Equal(t, mapped.Children[1].Children[0].Children[1].ItemID, "13")
	require.Equal(t, mapped.Children[1].Children[1].Children[0].ItemID, "14")
	require.Equal(t, mapped.Children[1].Children[1].Children[1].ItemID, "15")
}

func TestParseAndFormat(t *testing.T) {

	original := []Item{
		{ItemID: "1", ParentID: "", Name: "One"},
		{ItemID: "2", ParentID: "1", Name: "Two"},
		{ItemID: "3", ParentID: "1", Name: "Three"},
		{ItemID: "4", ParentID: "2", Name: "Four"},
		{ItemID: "5", ParentID: "2", Name: "Five"},
		{ItemID: "6", ParentID: "3", Name: "Six"},
		{ItemID: "7", ParentID: "3", Name: "Seven"},
		{ItemID: "8", ParentID: "4", Name: "Eight"},
		{ItemID: "9", ParentID: "4", Name: "Nine"},
		{ItemID: "10", ParentID: "5", Name: "Ten"},
		{ItemID: "11", ParentID: "5", Name: "Eleven"},
		{ItemID: "12", ParentID: "6", Name: "Twelve"},
		{ItemID: "13", ParentID: "6", Name: "Thirteen"},
		{ItemID: "14", ParentID: "7", Name: "Fourteen"},
		{ItemID: "15", ParentID: "7", Name: "Fifteen"},
	}

	formatted := ParseAndFormat(original)
	// test_debug(t, formatted)

	require.Equal(t, "1", formatted[0].ItemID)
	require.Equal(t, 0, formatted[0].Depth)

	require.Equal(t, "2", formatted[1].ItemID)
	require.Equal(t, 1, formatted[1].Depth)

	require.Equal(t, "4", formatted[2].ItemID)
	require.Equal(t, 2, formatted[2].Depth)

	require.Equal(t, "8", formatted[3].ItemID)
	require.Equal(t, 3, formatted[3].Depth)

	require.Equal(t, "9", formatted[4].ItemID)
	require.Equal(t, 3, formatted[4].Depth)

	require.Equal(t, "5", formatted[5].ItemID)
	require.Equal(t, 2, formatted[5].Depth)

	require.Equal(t, "10", formatted[6].ItemID)
	require.Equal(t, 3, formatted[6].Depth)

	require.Equal(t, "11", formatted[7].ItemID)
	require.Equal(t, 3, formatted[7].Depth)

	require.Equal(t, "3", formatted[8].ItemID)
	require.Equal(t, 1, formatted[8].Depth)

	require.Equal(t, "6", formatted[9].ItemID)
	require.Equal(t, 2, formatted[9].Depth)

	require.Equal(t, "12", formatted[10].ItemID)
	require.Equal(t, 3, formatted[10].Depth)

	require.Equal(t, "13", formatted[11].ItemID)
	require.Equal(t, 3, formatted[11].Depth)

	require.Equal(t, "7", formatted[12].ItemID)
	require.Equal(t, 2, formatted[12].Depth)

	require.Equal(t, "14", formatted[13].ItemID)
	require.Equal(t, 3, formatted[13].Depth)

	require.Equal(t, "15", formatted[14].ItemID)
	require.Equal(t, 3, formatted[14].Depth)
}

/*
func test_debug(t *testing.T, formatted []*Item) {
	t.Log("Formatted Items:")
	for _, item := range formatted {
		t.Logf("ItemID: %s, Depth: %d", item.ItemID, item.Depth)
	}
}
*/
