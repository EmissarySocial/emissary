behavior hotkey

	on keydown(key, metaKey, shiftKey, ctrlKey)

		set shortcut to ""

		if window.navigator.userAgent contains "Macintosh" then 
			if metaKey then 
				append "Ctrl+" to shortcut
			else
				exit
			end
		else 
			if ctrlKey then
				append "Ctrl+" to shortcut
			else
				exit
			end
		end

		if key.length > 1 then 
			exit
		end

		if shiftKey then
			append "Shift+" to shortcut
		end

		append key.toUpperCase() to shortcut
		set button to first <[aria-keyshortcuts="${shortcut}"] />

		if button is undefined then
			exit
		end

		halt the event
		send click to button
	end
end