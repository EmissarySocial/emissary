
behavior autosubmit

	on keydown(key, metaKey, shiftKey, ctrlKey)

		-- only autosubmit on plain ENTER key (no modifiers)
		if metaKey or shiftKey or ctrlKey then
			exit
		end

		-- autosubmit when ENTER key is pressed
		if key.toUpperCase() is "ENTER" then 
			halt the event

			-- only autosubmit on non-empty contents
			if my.innerHTML != "" then 
				send autosubmit to the closest <form/>
				set my.innerHTML to ""
			end
		end
