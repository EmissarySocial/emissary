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

	set url to location.href

	fetch `${url}` with method:"POST", headers:{"Content-Type": "application/json"}, body:body as JSON then
	reload() the window's location


behavior wysiwyg

init
	make a Quill from me, {theme:"bubble"} called quill
	log quill
