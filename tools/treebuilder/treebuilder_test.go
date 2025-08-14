package treebuilder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testItem struct {
	ItemID   string
	ParentID string
	Name     string
}

func (t testItem) TreeID() string {
	return t.ItemID
}

func (t testItem) TreeParent() string {
	return t.ParentID
}

func TestParse(t *testing.T) {

	original := []testItem{
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

	require.Equal(t, mapped.TreeID(), "1")

	require.Equal(t, mapped.Children[0].TreeID(), "2")
	require.Equal(t, mapped.Children[1].TreeID(), "3")

	require.Equal(t, mapped.Children[0].Children[0].TreeID(), "4")
	require.Equal(t, mapped.Children[0].Children[1].TreeID(), "5")
	require.Equal(t, mapped.Children[1].Children[0].TreeID(), "6")
	require.Equal(t, mapped.Children[1].Children[1].TreeID(), "7")

	require.Equal(t, mapped.Children[0].Children[0].Children[0].TreeID(), "8")
	require.Equal(t, mapped.Children[0].Children[0].Children[1].TreeID(), "9")
	require.Equal(t, mapped.Children[0].Children[1].Children[0].TreeID(), "10")
	require.Equal(t, mapped.Children[0].Children[1].Children[1].TreeID(), "11")
	require.Equal(t, mapped.Children[1].Children[0].Children[0].TreeID(), "12")
	require.Equal(t, mapped.Children[1].Children[0].Children[1].TreeID(), "13")
	require.Equal(t, mapped.Children[1].Children[1].Children[0].TreeID(), "14")
	require.Equal(t, mapped.Children[1].Children[1].Children[1].TreeID(), "15")
}

func TestParseAndFormat(t *testing.T) {

	original := []testItem{
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

	require.Equal(t, "1", formatted[0].TreeID())
	require.Equal(t, 0, formatted[0].Depth)

	require.Equal(t, "2", formatted[1].TreeID())
	require.Equal(t, 1, formatted[1].Depth)

	require.Equal(t, "4", formatted[2].TreeID())
	require.Equal(t, 2, formatted[2].Depth)

	require.Equal(t, "8", formatted[3].TreeID())
	require.Equal(t, 3, formatted[3].Depth)

	require.Equal(t, "9", formatted[4].TreeID())
	require.Equal(t, 3, formatted[4].Depth)

	require.Equal(t, "5", formatted[5].TreeID())
	require.Equal(t, 2, formatted[5].Depth)

	require.Equal(t, "10", formatted[6].TreeID())
	require.Equal(t, 3, formatted[6].Depth)

	require.Equal(t, "11", formatted[7].TreeID())
	require.Equal(t, 3, formatted[7].Depth)

	require.Equal(t, "3", formatted[8].TreeID())
	require.Equal(t, 1, formatted[8].Depth)

	require.Equal(t, "6", formatted[9].TreeID())
	require.Equal(t, 2, formatted[9].Depth)

	require.Equal(t, "12", formatted[10].TreeID())
	require.Equal(t, 3, formatted[10].Depth)

	require.Equal(t, "13", formatted[11].TreeID())
	require.Equal(t, 3, formatted[11].Depth)

	require.Equal(t, "7", formatted[12].TreeID())
	require.Equal(t, 2, formatted[12].Depth)

	require.Equal(t, "14", formatted[13].TreeID())
	require.Equal(t, 3, formatted[13].Depth)

	require.Equal(t, "15", formatted[14].TreeID())
	require.Equal(t, 3, formatted[14].Depth)
}
