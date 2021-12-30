behavior wysiwyg(name)

	-- Assemble the wysiwyg editor
	init 
		append `<input type=hidden name="${name}">` to me 
		
		set editor to the first <.wysiwyg-editor /> in me
		if editor is null
			append `<div class="wysiwyg-editor"></div>` to me
			set editor to the first <.wysiwyg-editor /> in me
		end

		tell <button/> in me
			add [@type="button"]
		end

		add [@tabIndex=0] to the editor
		add [@contentEditable=true] to the editor
	end


	-- All toolbar options handled here
	on click
		set command to the target's [@data-command]

		if command is not null then 
			set value to the target's [@data-command-value]
			call document.execCommand(command, false, value)
		end
	end

	-- Save on blur, or every 10 seconds
	on blur or keyup debounced at 10s
		set node to document.getElementById(id)
		if node == null then
			log "error, node: " + id + " is undefined."
			exit
		end

		set the node's value to my innerHTML
		trigger save
	end
end


behavior hotkey

	init
		set value to "b"
		set range to <[data-hotkey]/> in me
		log range

		set node to <[data-hotkey="${value}"]/>
		log node
	end

	on keydown(key, metaKey, ctrlKey)

		if (metaKey and ctrlKey) == false then 
			exit
		end

		if key.length > 1 then 
			exit
		end

		set button to <[data-hotkey="${key}"] />

		if button is null then
			exit
		end

		trigger click on button
		halt the event
	end
end