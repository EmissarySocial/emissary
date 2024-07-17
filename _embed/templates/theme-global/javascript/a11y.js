/** 
 * This file adds the "accessibility" extension to htmx, which scans 
 * all content for elements that *should* be focusable, then adds 
 * attributes to guarantees that the browser will focus on them and
 * keyboard event handlers that work as an alternative for "clicks"
 */

(function(){

	var api;

	htmx.defineExtension("a11y", {

		init: function(internalAPI) {
			api = internalAPI
		},

		onEvent: function(/** @type {string} */ name, /** @type {Event} */ event) {

			switch (name) {

				// Guarantee that: 1) interactive elements are focusable,
				// and 2) targeted elements are live regions
				case "htmx:afterProcessNode":

				var element = event.target

				if (event.detail != null) {
					if (event.detail.elt != null) {
						element = event.detail.elt
					}
				}
	
				if (element == null) {
					return
				}
	
				// Special rules for links and buttons
				element.querySelectorAll("a,button,[role=link],[role=button],[role=tab]").forEach(function(/** @type {HTMLElement} */ node) {
					
					// If tabIndex is not already set, then default it to 0
					if (node.attributes["tabIndex"] == undefined) {
						node.tabIndex = 0
					}
	
					// If node is focusable (and not already a link or button) then add keyboard handlers for ENTER and SPACE keys
					if (node.tabIndex != -1) {
						node.addEventListener("keyup", function(event) {
							if (event.key == "Enter") {
								htmx.trigger(node, "click")
							}
						})
					}
				})
	
				// Scan for hx-target attributes and add `aria-live="polite"` to any targeted elements
				element.querySelectorAll("[hx-target],[data-hx-target]").forEach(function(/** @type {HTMLElement} */ node) {
					var target = api.getTarget(node)
					if (target.attributes["aria-live"] == undefined) {
						target.setAttribute("aria-live", "polite")
					}
				})
				break

				// Focus on new content after swapping it in
				case "htmx:afterSwap":
					if (event.target == null)  {
						return
					}

					// event.target.tabIndex = "-1"
					// event.target.focus()
					break;
			}
		}
	})
})();