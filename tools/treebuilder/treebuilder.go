package treebuilder

import (
	"github.com/davecgh/go-spew/spew"
)

func ParseAndFormat(items []Item) []*Item {

	defer func() {
		if err := recover(); err != nil {
			spew.Dump(err)
		}
	}()

	tree := Parse(items)
	result := make([]*Item, 0, len(items))
	result = format(tree, result)

	return result
}

// Parse links all items in the slice, as `Children`
// of the root node, returning the root node
func Parse(items []Item) *Item {

	var rootNode *Item

	hash := hashmap(items)

	for index, item := range items {

		if item.ParentID == "" {
			if rootNode == nil {
				items[index].Depth = 0
				rootNode = &items[index]
			} else {
				spew.Dump("ParentID is empty... false root -------------------------", item)
			}
			continue
		}

		if parent := hash[item.ParentID]; parent != nil {
			items[index].Depth = parent.Depth + 1
			parent.Children = append(parent.Children, &items[index])
		} else {
			spew.Dump("Parent not found -------------------------", item)
		}
	}

	return rootNode
}

// format generates a new slice of pointers in tree order
// that each identifies the next tree item to display
func format(item *Item, result []*Item) []*Item {

	result = append(result, item)

	for _, child := range item.Children {
		result = format(child, result)
	}

	return result
}
