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
		call _hyperscript.processNode(#modal)


--------------------------------
-- WYSIWYG Editor

behavior wysiwyg

init

	-- get editor config
	fetch "/static/editor/wysiwyg.json" as json
	put it into config

	-- initialize ckEditor
	set editorNode to first <.ck-editor/> in me
	set my editor to InlineEditor.create(editorNode, config)

	-- initialize the htmx form
	set @hx-post to @action
	set @hx-target to "#toaster"
	set @hx-swap to "innerHTML"
	set @hx-trigger to "save"
	set @hx-push-url to false
	call htmx.process(me)

	set hidden to first <[name=html]/> in me
	set hidden@value to my editor.getData()

on blur from <.ck-editor/>

	set hidden to first <[name=html]/> in me
	if my editor.getData() is not hidden@value then 
		set hidden@value to my editor.getData()
		trigger save
	end

-- need a better way of detecting a page change.
-- on htmx:beforeSwap from window
--	destroy() my editor
