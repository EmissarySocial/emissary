--------------------------------
-- Helper Functions

def getMyPosition(node, class)

	set siblings to node's parentNode's children
	set index to 0

	repeat for sibling in siblings
		if class is null or sibling[@class] is class then 
			if sibling is node then 
				return index	
			end
			increment index 
		end
	end

	return 0

def getContainerType(node) 

	set container to closest <.container /> to node
	return container[@container-style]


--------------------------------
-- Containers

behavior containerInsertPoint

	init
		add .container-insert-point

	on click
		set container to closest .container
		set parentId to container[@data-id]
		set style to container[@data-style]
		set check to container[@data-check]
		set childIndex to getMyPosition(me, "container-insert-point")

		set url to the location's href
		set url to url.replace("/draft", `/layout/contentEditor-addItem?style=${style}&parentId=${parentId}&childIndex=${childIndex}&check=${check}`)

		fetch `${url}`
		put it at end of document's body
		call htmx.process(#modal)


--------------------------------
-- WYSIWYG Editor

behavior wysiwygForm

	init
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
		trigger save		

behavior wysiwygEditor

	init 
		fetch "/static/editor/wysiwyg.json" as json
		put it into config
		set editor to InlineEditor.create(me, config)
		repeat forever
			wait for blur
			set editor.isReadOnly to true
			send beforeSave(html:editor.getData()) to closest <form/>
			wait for htmx:afterOnLoad from window
			set editor.isReadOnly to false
		end

