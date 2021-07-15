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

behavior containerInsert

	init
		add .container-insert

	on click

		set body to {
			"type": "new-item",
            "itemType": "WYSIWYG",
            "itemId": @data-itemId,
            "place": @data-place,
            "check": @data-check
		}

		set url to the location's href
 
		fetch `${url}` {method:"POST", headers:{"Content-Type": "application/json"}, body: body as JSON}
		reload() the window's location

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
