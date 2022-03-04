behavior wysiwyg(name)

	-- WYSIWYG setup
	init 
		-- save links to important DOM nodes
		set element form to closest <form />
		set element input to form.elements[name]
		set element editor to first <.wysiwyg-editor /> in me

		-- configure related DOM nodes
		add [@tabIndex=0] to element editor
		add [@contentEditable=true] to element editor

		tell <button/> in me
			add [@type="button"]
		end

	-- Clicking a toolbar button triggers a command on the content
	on click(target)

		if target's [@data-command] is null then 
			set target to closest <[data-command]/> to target
			if target is null then
				exit
			end
		end

		set command to target's [@data-command]

		-- special handling for inertLink
		if command is "createLink" then
			get prompt("Enter Link URL")
			call document.execCommand(command, false, result)
			exit
		end

		-- fall through to all other commands
		set value to target's [@data-command-value]
		call document.execCommand(command, false, value)
	end

	-- Show the toolbar when focused
	on focus(target) from <.wysiwyg-editor /> in me

		tell <.wysiwyg-toolbar /> in me
			remove [@hidden]
		end
	end

	-- Hide the toolbar when blured
	on blur from <.wysiwyg-editor /> in me

		wait 200ms
		if (<:focus/> in me) is empty then
			tell <.wysiwyg-toolbar /> in me
				add [@hidden=true]
			end
		end
	end

	-- Autosave the WYSIWYG after 15s of inactivity
	on input debounced at 15s
		send updated to form
	end
	
	-- Autosave the WYSIWYG whenever it loses focus
	on blur from <.wysiwyg-editor />
		send updated to form
	end

	-- Push the value directly into the XHR request before it's sent.
	on htmx:configRequest(parameters) from closest <form/>
		set value to the editor's innerHTML
		Object.defineProperty(parameters, name, {value: value, writable:'true'})
	end