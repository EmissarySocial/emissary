package treebuilder

type Item struct {
	ItemID    string
	ParentID  string
	URL       string
	Name      string
	Content   string
	Icon      string
	Published int64
	Depth     int
	Sort      int
	Children  []*Item
}

func NewItem() Item {

	return Item{
		Children: make([]*Item, 0),
	}
}

func (item Item) ID() string {
	return item.ItemID
}
