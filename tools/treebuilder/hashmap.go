package treebuilder

// hashmap makes a hash of items in the input slice
func hashmap[T TreeGetter](items []T) map[string]*Tree[T] {
	result := make(map[string]*Tree[T], len(items))

	for index := range items {
		treeID := items[index].TreeID()
		result[treeID] = NewTree(items[index])
	}

	return result
}
