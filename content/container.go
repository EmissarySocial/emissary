package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeContainer = "CONTAINER"

const ContainerStyleRows = "ROWS"

const ContainerStyleColumns = "COLS"

const ContainerPlaceAbove = "ABOVE"

const ContainerPlaceBelow = "BELOW"

const ContainerPlaceLeft = "LEFT"

const ContainerPlaceRight = "RIGHT"

type Container struct{}

func (widget Container) View(builder *html.Builder, content Content, id int) {
	item := content.GetItem(id)

	builder.Div().
		Class("container").
		Data("style", item.GetString("style")).
		Data("size", strconv.Itoa(len(item.Refs)))

	for _, index := range item.Refs {
		builder.Div().Class("container-item")
		content.viewSubTree(builder, index)
		builder.Close()
	}
}

func (widget Container) Edit(builder *html.Builder, content Content, id int, endpoint string) {

	item := content.GetItem(id)
	idString := strconv.Itoa(id)
	style := item.GetString("style")

	builder.Div().
		Class("container").
		Data("style", style).
		Data("size", strconv.Itoa(len(item.Refs))).
		Data("id", idString)

	// For containers with multiple items, add an insertion point that cross-cuts the beginning of the container
	if id == 0 {
		builder.Div().Script("install containerInsert").Data("itemId", idString).Data("place", ContainerPlaceLeft).Data("check", item.Check).Close()
		builder.Div().Script("install containerInsert").Data("itemId", idString).Data("place", ContainerPlaceAbove).Data("check", item.Check).Close()
	}

	for childIndex, childID := range item.Refs {
		childIDString := strconv.Itoa(childID)
		builder.Div().Class("container-item")

		if widget.showInsertMarker(content, id, childIndex, ContainerPlaceAbove) {
			builder.Div().Script("install containerInsert").Data("itemId", childIDString).Data("place", ContainerPlaceAbove).Data("check", item.Check).Close()
		}

		if widget.showInsertMarker(content, id, childIndex, ContainerPlaceLeft) {
			builder.Div().Script("install containerInsert").Data("itemId", childIDString).Data("place", ContainerPlaceLeft).Data("check", item.Check).Close()
		}

		content.editSubTree(builder, childID, endpoint)

		if widget.showInsertMarker(content, id, childIndex, ContainerPlaceRight) {
			builder.Div().Script("install containerInsert").Data("itemId", childIDString).Data("place", ContainerPlaceRight).Data("check", item.Check).Close()
		}

		if widget.showInsertMarker(content, id, childIndex, ContainerPlaceBelow) {
			builder.Div().Script("install containerInsert").Data("itemId", childIDString).Data("place", ContainerPlaceBelow).Data("check", item.Check).Close()
		}
		builder.Close()
	}

	// For containers with multiple items, add an insertion point the cross-cuts the end of the container
	if id == 0 {
		builder.Div().Script("install containerInsert").Data("itemId", idString).Data("place", ContainerPlaceRight).Data("check", item.Check).Close()
		builder.Div().Script("install containerInsert").Data("itemId", idString).Data("place", ContainerPlaceBelow).Data("check", item.Check).Close()
	}

	builder.Close()
}

func (widget Container) Default() Item {
	return Item{}
}

/*
func (widget Container) insertMarker(builder *html.Builder, content Content, parentID int, childIndex int, place string) {

	if showInsertMarker(content, parentID, childIndex, place) {
		builder.Div().Script("install containerInsert").Data("itemId", childIDString).Data("place", place).Data("check", item.Check).Close()
	}
}
*/

// showInsertMarker returns TRUE if an container insertion marker should be shown at this location
func (widget Container) showInsertMarker(content Content, parentID int, childIndex int, place string) bool {

	switch content[parentID].GetString("style") {

	case ContainerStyleRows:

		switch place {
		case ContainerPlaceAbove:
			return childIndex > 0
		case ContainerPlaceBelow:
			return false
		case ContainerPlaceLeft:
			return true
		case ContainerPlaceRight:
			return true
		}

	case ContainerStyleColumns:

		switch place {
		case ContainerPlaceLeft:
			return childIndex > 0
		case ContainerPlaceRight:
			return false
		case ContainerPlaceAbove:
			return true
		case ContainerPlaceBelow:
			return true
		}
	}

	return false
}
