package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeWYSIWYG = "WYSIWYG"

func WYSIWYGViewer(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("html")
	b.WriteString(result)
}

func WYSIWYGEditor(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("html")
	idString := strconv.Itoa(id)
	path := "id-" + idString

	formScript := `
	on load 
		set @hx-post to @action
		set @hx-target to "#toaster"
		set @hx-swap to "innerHTML"
		set @hx-trigger to "save"
		set @hx-push-url to false
		call htmx.process(me)

	on beforeSave(html)
		tell <[name=html]/> in me
			set @value to html
		end
		trigger save`

	wysiwygScript := `
	on load 
		set editor to InlineEditor.create(me, window.wysiwygConfig)
		repeat forever
			wait for blur
			set editor.isReadOnly to true
			send beforeSave(html:editor.getData()) to closest <form/>
			wait for htmx:afterOnLoad from window
			set editor.isReadOnly to false
		end
	end`

	b.Form("post", lib.Endpoint).ID(path + "-form").Script(formScript)
	{
		b.Input("hidden", "type").Value("update-item")
		b.Input("hidden", "itemId").Value(idString)
		b.Input("hidden", "hash").Value(item.Hash)
		b.Input("hidden", "html")
		b.Div().ID(path).Class("ck-editor editor-widget").Script(wysiwygScript).InnerHTML(result)
	}

	b.CloseAll()

	b.CloseAll()
}
