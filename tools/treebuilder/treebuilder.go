package treebuilder

import (
	"github.com/benpate/derp"
)

func ParseAndFormat[T TreeGetter](items []T) []*Tree[T] {

	defer func() {
		if err := recover(); err != nil {
			derp.Report(derp.InternalError("treebuilder.ParseAndFormat", "Unknown panic in ParseAndFormat", err))
		}
	}()

	tree := Parse(items)
	result := make([]*Tree[T], 0, len(items))
	result = Format(tree, result)

	return result
}

// Parse links all trees in the slice, as `Children`
// of the root node, returning the root node
func Parse[T TreeGetter](items []T) *Tree[T] {

	var rootNode *Tree[T]

	hash := hashmap(items)

	for _, item := range items {

		treeID := item.TreeID()
		parentID := item.TreeParent()

		if parentID == "" {
			if rootNode == nil {
				hash[treeID] = NewTree(item)
				rootNode = hash[treeID]
			}
			continue
		}

		if parent := hash[parentID]; parent != nil {
			hash[treeID] = NewTree(item)
			hash[treeID].Depth = parent.Depth + 1
			parent.Children = append(parent.Children, hash[treeID])
		}
	}

	return rootNode
}

// Format generates a new slice of pointers in tree order
// that each identifies the next tree tree to display
func Format[T TreeGetter](tree *Tree[T], result []*Tree[T]) []*Tree[T] {

	result = append(result, tree)

	for _, child := range tree.Children {
		result = Format(child, result)
	}

	return result
}
