package content

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

type Item struct {
	Type string       `json:"type" bson:"type"`
	Kids []Item       `json:"kids" bson:"kids"`
	Data datatype.Map `json:"data" bson:"data"`
}

// Surface setters from the data struct.
func (item *Item) GetString(key string) string {
	return item.Data.GetString(key)
}

func (item *Item) GetInt(key string) int {
	return item.Data.GetInt(key)
}

func (item *Item) GetInterface(key string) interface{} {
	return item.Data.GetInterface(key)
}

// GetPath implemnts the path.Getter interface
func (item *Item) GetPath(p path.Path) (interface{}, error) {

	head, tail := p.Split()

	if index, err := p.Index(len(item.Kids)); err == nil {
		return tail.Get(item.Kids[index])
	}

	if head == "type" {
		return tail.Get(item.Type)
	}

	return p.Get(item.Data)
}

// SetPath implements the path.Setter interface
func (item *Item) SetPath(p path.Path, value interface{}) error {

	head, tail := p.Split()

	if head == "type" {
		return derp.New(500, "content.Item.SetPath", "Cannot change Item.Type", value)
	}

	if index, err := p.Index(len(item.Kids)); err == nil {
		return tail.Set(&(item.Kids[index]), value)
	}

	return p.Set(item.Data, value)
}

// DeletePath implements the path.Deleter interface
func (item *Item) DeletePath(p path.Path) error {
	return nil
}
