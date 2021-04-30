package content

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/benpate/datatype"
	"github.com/benpate/path"
)

// Item represents a single piece of content.  It will be rendered by one of several rendering
// Libraries, using the custom data it contains.
type Item struct {
	Type string       `json:"type" bson:"type"`
	Refs []int        `json:"refs" bson:"refs"`
	Data datatype.Map `json:"data" bson:"data"`
	Hash string       `json:"hash" bson:"hash"` // A random code or nonce to authenticate requests
}

// NewItem returns a fully initialized Item
func NewItem(t string) Item {
	result := Item{
		Type: t,
		Data: make(datatype.Map),
	}

	result.NewHash()
	return result
}

// NewHash updates the hash value for this item
func (item *Item) NewHash() {
	seed := time.Now().Unix()
	source := rand.NewSource(seed)
	item.Hash = strconv.FormatInt(source.Int63(), 36) + strconv.FormatInt(source.Int63(), 36)
}

// Surface setters from the data struct.
func (item *Item) GetString(key string) string {
	return item.Data.GetString(key)
}

func (item *Item) GetInt(key string) int {
	return item.Data.GetInt(key)
}

func (item *Item) GetSliceOfInt(key string) []int {
	return item.Data.GetSliceOfInt(key)
}

func (item *Item) GetSliceOfString(key string) []string {
	return item.Data.GetSliceOfString(key)
}

func (item *Item) GetInterface(key string) interface{} {
	return item.Data.GetInterface(key)
}

// GetPath implemnts the path.Getter interface
func (item *Item) GetPath(p path.Path) (interface{}, error) {
	return item.Data.GetPath(p)
}

// SetPath implements the path.Setter interface
func (item *Item) SetPath(p path.Path, value interface{}) error {
	return item.Data.SetPath(p, value)
}

// DeletePath implements the path.Deleter interface
func (item *Item) DeletePath(p path.Path) error {
	// return item.Data.DeletePath(p)
	return nil
}

// AddReference adds a new "sub-item" reference to this item
func (item *Item) AddReference(to int) {

	// first, verify that we don't already have
	// a ref to this same item
	for index := range item.Refs {
		if item.Refs[index] == to {
			return
		}
	}

	// fall through means we can add a new ref.
	item.Refs = append(item.Refs, to)
}

// UpdateReference migrates references from an old value to a new one
func (item *Item) UpdateReference(from int, to int) {
	for index := range item.Refs {
		if item.Refs[index] == from {
			item.Refs[index] = to
			return
		}
	}
}

// DeleteReference removes a reference from this Item.
func (item *Item) DeleteReference(id int) {
	for index := range item.Refs {
		if item.Refs[index] == id {
			item.Refs = append(item.Refs[:index], item.Refs[index+1:]...)
			return
		}
	}
}
