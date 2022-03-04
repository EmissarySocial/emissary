
behavior Autosubmit

	on keydown(key, metaKey, shiftKey, ctrlKey)

		if metaKey or shiftKey or ctrlKey then
			exit
		end

		if key.toUpperCase() is "ENTER" then 
			halt the event
			send submitForm to the closest <form/>
			set my.innerHTML to ""
		end
