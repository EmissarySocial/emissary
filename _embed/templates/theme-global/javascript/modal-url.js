/**
 * This file restores the address bar when a modal that wrote its own URL closes.
 *
 * A modal step with `replace-url` (as-modal) answers with an HX-Replace-Url header, so the
 * address bar shows the modal's own, shareable URL while it is open.  This remembers the
 * address the page had before the first such modal opened, and writes it back when the
 * modal closes.  Swapping one modal for another (the next photo in a gallery) keeps the
 * address from before the first one.
 *
 * A modal opened directly, as a full page, names the address of the page behind it in
 * data-return-url, and that is the address written back.
 *
 * The address is written back only while it is still the one the modal wrote.  When
 * something inside the modal moved the page somewhere else, the page's address is left
 * alone.
 */

(function() {

	var returnURL = null   // The page's address before the first modal replaced it
	var modalURL = null    // The address the open modal last wrote

	// Before htmx replaces the address for a response bound for <aside>, remember the
	// page's own address, unless a modal is already open and has done so already
	document.addEventListener("htmx:beforeHistoryUpdate", function(event) {

		var detail = event.detail || {}
		var history = detail.history || {}

		if (history.type !== "replace") {
			return
		}

		if (!(detail.target instanceof Element) || detail.target.tagName !== "ASIDE") {
			return
		}

		if (returnURL === null) {
			returnURL = window.location.href
		}

		modalURL = new URL(history.path, window.location.href).href
	})

	// A modal opened directly, on a full page drawn over another page's address (as-modal's
	// background-url), names that address in data-return-url
	function readReturnURL() {

		var modal = document.querySelector("#modal[data-return-url]")

		if (modal === null) {
			return
		}

		returnURL = new URL(modal.getAttribute("data-return-url"), window.location.href).href
		modalURL = window.location.href
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", readReturnURL)
	} else {
		readReturnURL()
	}

	// When the modal closes, put the page's own address back
	window.addEventListener("closeModal", function() {

		if (returnURL === null) {
			return
		}

		if (window.location.href === modalURL) {
			window.history.replaceState(window.history.state, "", returnURL)
		}

		returnURL = null
		modalURL = null
	})

})();
