/** 
 * This file adds the "accessibility" extension to htmx, which scans 
 * all content for elements that *should* be focusable, then adds 
 * attributes to guarantees that the browser will focus on them and
 * keyboard event handlers that work as an alternative for "clicks"
 */

htmx.defineExtension("a11y", {

	init: function() {},

	onEvent: function(/** @type {string} */ name, /** @type {Event} */ event) {

		// Only take actions on "htmx:afterProcessNode"
		if (name !== "htmx:afterProcessNode") {
			return;
		}

		// Special rules for links and buttons
		event.target.querySelectorAll("a,[role=link],button,[role=button]").forEach(function(/** @type {HTMLElement} */ node) {
			
			// If tabIndex is not already set, then default it to 0
			if (node.attributes["tabIndex"] == undefined) {
				node.tabIndex = 0
			}

			// If this node is focusable, then add keyboard event handlers...
			if (node.tabIndex != -1) {
				node.addEventListener("keyup", function(event) {
					if ((event.key == "Enter") || (event.key == " ")) {
						htmx.trigger(node, "click")
					}
				})
			}
		})
	}
})
