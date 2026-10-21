/**
 * copyToClipboard writes text to the clipboard, for the Clipboard hyperscript behavior.
 * navigator.clipboard exists only in a secure context (https or localhost), so a site served
 * over plain http falls back to selecting a hidden textarea and running execCommand("copy").
 */
window.copyToClipboard = function(text) {

	if (navigator.clipboard) {
		return navigator.clipboard.writeText(text)
	}

	var textarea = document.createElement("textarea")
	textarea.value = text
	textarea.setAttribute("readonly", "")
	textarea.style.position = "fixed"
	textarea.style.opacity = "0"
	document.body.appendChild(textarea)
	textarea.select()

	var copied = document.execCommand("copy")
	document.body.removeChild(textarea)

	if (!copied) {
		return Promise.reject(new Error("Unable to copy to the clipboard"))
	}

	return Promise.resolve()
}
