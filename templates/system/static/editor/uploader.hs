behavior Uploader(url, success)

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
		call formData.append("file", file)
		fetch `${url}` {method:"POST", body:formData} as text
	end

	set the window's location to success

end

