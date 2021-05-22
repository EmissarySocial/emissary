package content

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/benpate/html"
)

// Content represents a complete package of content
type Content []Item

func (content Content) View() string {
	builder := html.New()
	widget := content.Widget(0)

	widget.View(builder, content, 0)
	return builder.String() // + "<pre>" + spew.Sdump(content) + "</pre>"
}

func (content *Content) Widget(id int) Widget {

	// Bounds check
	if (id < 0) || (id >= len(*content)) {
		return Nil{}
	}

	itemType := (*content)[id].Type

	switch itemType {

	case ItemTypeContainer:
		return Container{}
	case ItemTypeHTML:
		return HTML{}
	case ItemTypeOEmbed:
		return OEmbed{}
	case ItemTypeTabs:
		return Tabs{}
	case ItemTypeText:
		return Text{}
	case ItemTypeWYSIWYG:
		return WYSIWYG{}
	default:
		return Nil{}
	}
}

func (content Content) Edit(endpoint string) string {
	builder := html.New()
	widget := content.Widget(0)

	widget.Edit(builder, content, 0, endpoint)
	return builder.String() // + "<pre>" + spew.Sdump(content) + "</pre>"
}

func (content Content) viewSubTree(builder *html.Builder, id int) {
	subBuilder := builder.SubTree()
	widget := content.Widget(id)

	widget.View(subBuilder, content, id)
	subBuilder.CloseAll()
}

func (content Content) editSubTree(builder *html.Builder, id int, endpoint string) {
	subBuilder := builder.SubTree()
	widget := content.Widget(id)

	widget.Edit(subBuilder, content, id, endpoint)
	subBuilder.CloseAll()
}

// AddItem adds a new item to this content structure, and returns the new item's index
func (content *Content) AddItem(item Item) int {
	newID := len(*content)

	*content = append(*content, item)

	return newID
}

// GetItem returns a pointer to the item at the desired index
func (content Content) GetItem(id int) *Item {
	return &(content[id])
}

func (content Content) GetParent(id int) (int, *Item) {

	for itemIndex := range content {
		for refIndex := range content[itemIndex].Refs {
			if content[itemIndex].Refs[refIndex] == id {
				return itemIndex, &(content[itemIndex])
			}
		}
	}

	return -1, nil
}

// Compact removes any unused items in the content slice
// and reorganizes references
func (content *Content) Compact() {
	front := 0
	back := len(*content) - 1

	for front < back {

		if (*content)[front].Type != "" {
			front = front + 1
			continue
		}

		if (*content)[back].Type == "" {
			back = back - 1
			continue
		}

		content.move(back, front)
	}

	if (*content)[back].Type != "" {
		back = back - 1
	}

	*content = (*content)[:back]
}

// move physically moves an item from one index to another (overwriting the target location)
// and updates references
func (content Content) move(from int, to int) {

	content[to] = content[from]
	content[from] = Item{}

	for index := range content {
		content[index].UpdateReference(from, to)
	}
}

// NewChecksum generates a new checksum value to be inserted into a content.Item
func NewChecksum() string {
	seed := time.Now().Unix()
	source := rand.NewSource(seed)
	return strconv.FormatInt(source.Int63(), 36) + strconv.FormatInt(source.Int63(), 36)
}
