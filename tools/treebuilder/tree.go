package treebuilder

type Tree[T TreeGetter] struct {
	Getter   T
	Depth    int
	Children []*Tree[T]
}

func NewTree[T TreeGetter](getter T) *Tree[T] {
	result := Tree[T]{
		Getter:   getter,
		Children: make([]*Tree[T], 0),
	}

	return &result
}

func (tree *Tree[T]) TreeID() string {
	return tree.Getter.TreeID()
}

func (tree *Tree[T]) ParentID() string {
	return tree.Getter.TreeParent()
}
