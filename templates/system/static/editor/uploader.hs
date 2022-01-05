behavior DropToUpload
	
on click(target)
	set input to the first <input[type="file"]/> in me then 
	if target is not input then
		send click to input
	end

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

	set input to the first <input[type="file"]/> in me
	set the input's files to the dataTransfer's files

on htmx:xhr:progress
	log event
end