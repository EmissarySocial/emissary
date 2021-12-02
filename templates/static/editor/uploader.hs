behavior Uploader(url, accept)

init
	set result to ""
	append `Drag Files Into This Box, or Click to Choose Files`
	append `<form hx-post="${url}" hx-encoding="multipart/form-data" style="display:none;">`
	append `<input type="file" name="files" accept="${accept}" multiple/>`
	append `</form>`
	put it at the end of me
	call htmx.process(me)

on click 
	send click to the first <input[type="file"]/> in me

on change 
	log "got it"

on dragenter
	halt the event
	add .highlight

on dragover
	halt the event
	add .highlight

on dragleave
	halt the event
	remove .highlight

on drop(dataTransfer)
	halt the event
	remove .highlight

	for file in dataTransfer.files
		make a FormData called formData
		call formData.append("files", file)
	end

	fetch `${url}` {method:"POST", body:formData} as text
	set the window's location to the `${result}`

end