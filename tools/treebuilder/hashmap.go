package treebuilder

import "github.com/davecgh/go-spew/spew"

// hashmap makes a hash of items in the input slice
func hashmap(items []Item) map[string]*Item {
	result := make(map[string]*Item, len(items))

	spew.Dump("hashmap ---------------------------------------")
	for i := range items {
		spew.Dump(items[i].ItemID)
		result[items[i].ItemID] = &items[i]
	}
	spew.Dump("<<<<<<<<<<<")

	return result
}
