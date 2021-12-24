function makeWYSIWYG(node) {

	var quill = new Quill(node, {
		"theme": "bubble"
	});

	quill.focus();

	quill.on("selection-change", function() {
		if (!quill.hasFocus()) {
			var form = node.closest("form")
			var element = form.querySelector("input[name='html']");
			var delta = quill.getContents();
			console.log(delta)
			console.log(quill.root.innerHTML)
			element.value = quill.root.innerHTML;
			htmx.trigger(node, "quill:blur");
		}
	})

	console.log(quill)
}
