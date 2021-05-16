package content

import (
	"strconv"

	"github.com/benpate/datatype"
	"github.com/benpate/html"
)

const ItemTypeTabs = "TABS"

type Tabs struct{}

func (widget Tabs) View(builder *html.Builder, content Content, id int) {
	item := content.GetItem(id)
	labels := item.GetSliceOfString("labels")

	builder.Div().Class("tabs")
	for index, id := range item.Refs {
		nodeID := "#id-" + strconv.Itoa(id)
		label := labels[index]
		builder.A(nodeID).Class("tabs-label").InnerHTML(label).Close()
	}

	for _, id := range item.Refs {
		nodeID := "id-" + strconv.Itoa(id)
		builder.Div().ID(nodeID).EndBracket()
		content.viewSubTree(builder, id)
		builder.Close()
	}

	builder.Close()
}

func (widget Tabs) Edit(builder *html.Builder, content Content, id int, endpoint string) {
	item := content.GetItem(id)
	labels := item.GetSliceOfString("labels")

	builder.Div().Class("tabs")
	for index, id := range item.Refs {
		nodeID := "#id-" + strconv.Itoa(id)
		label := labels[index]
		builder.A(nodeID).Class("tabs-label").InnerHTML(label).Close()
	}

	for _, id := range item.Refs {
		nodeID := "id-" + strconv.Itoa(id)
		builder.Div().ID(nodeID).EndBracket()
		content.editSubTree(builder, id, endpoint)
		builder.Close()
	}

	builder.Close()
}

func (widget Tabs) DefaultChildren() []Item {
	return []Item{
		{
			Type: "CONTAINER",
			Data: datatype.Map{
				"style": "COLUMNS",
			},
		},
		{
			Type: "CONTAINER",
			Data: datatype.Map{
				"style": "COLUMNS",
			},
		},
	}
}
