package treebuilder

type TreeGetter interface {
	TreeID() string
	TreeParent() string
}
