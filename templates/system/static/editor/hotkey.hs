behavior hotkey

	on keydown(key, metaKey, ctrlKey)

		if window.navigator.userAgent contains "Macintosh" then 
			if metaKey is not true then 
				exit
			end
		else 
			if ctrlKey is not true then
				exit
			end
		end

		if key.length > 1 then 
			exit
		end

		set button to <[data-hotkey="${key}"] />

		if button is undefined then
			exit
		end

		trigger click on button
		halt event
	end
end