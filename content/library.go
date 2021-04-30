package content

import (
	"github.com/benpate/html"
)

type Widget func(*Library, *html.Builder, *PathMaker, *Item)

type Library struct {
	widgets map[string]Widget
}

func NewLibrary() Library {
	return Library{
		widgets: make(map[string]Widget),
	}
}

// ViewerLibrary generates a fully populated library
// containing all of the default controls.
func ViewerLibrary() Library {
	result := NewLibrary()

	result.Register(ItemTypeText, TextViewer)
	result.Register(ItemTypeWYSIWYG, WYSIWYGViewer)
	result.Register(ItemTypeHTML, HTMLViewer)
	result.Register(ItemTypeOEmbed, OEmbedViewer)
	result.Register(ItemTypeContainer, ContainerViewer)
	result.Register(ItemTypeTabs, TabsViewer)

	return result
}

func EditorLibrary() Library {
	result := NewLibrary()

	result.Register(ItemTypeText, TextEditor)
	result.Register(ItemTypeWYSIWYG, WYSIWYGEditor)
	result.Register(ItemTypeHTML, HTMLEditor)
	result.Register(ItemTypeOEmbed, OEmbedEditor)
	result.Register(ItemTypeContainer, ContainerEditor)
	result.Register(ItemTypeTabs, TabsEditor)

	return result
}

///////////////////////////////
// Library Methods

func (library *Library) Register(class string, widget Widget) *Library {
	library.widgets[class] = widget
	return library
}

// Render returns the HTML for a specific content.Item, based on the RenderType requested
func (library *Library) Render(item *Item) string {

	builder := html.New()
	pm := NewPathMaker()

	if widget, ok := library.widgets[item.Type]; ok {
		widget(library, builder, &pm, item)
	}

	return builder.String()
}

// RenderToBuilder uses the widget library to safely append values to an existing html.Builder
func (library *Library) SubTree(builder *html.Builder, pm *PathMaker, item *Item) {

	if widget, ok := library.widgets[item.Type]; ok {
		subBuilder := builder.SubTree()
		subPathMaker := pm.SubTree()
		widget(library, subBuilder, &subPathMaker, item)
		subBuilder.CloseAll()
	}
}
