package content

import (
	"github.com/benpate/html"
)

type Widget func(*Library, *html.Builder, Content, int)

type Library struct {
	widgets  map[string]Widget
	Endpoint string
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

func EditorLibrary(endpoint string) Library {
	result := NewLibrary()
	result.Endpoint = endpoint

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
func (library *Library) Render(content Content, id int) string {

	builder := html.New()

	if widget, ok := library.widgets[content[id].Type]; ok {
		widget(library, builder, content, id)
	}

	return builder.String()
}

// RenderToBuilder uses the widget library to safely append values to an existing html.Builder
func (library *Library) SubTree(builder *html.Builder, content Content, id int) {

	if widget, ok := library.widgets[content[id].Type]; ok {
		subBuilder := builder.SubTree()
		widget(library, subBuilder, content, id)
		subBuilder.CloseAll()
	}
}
