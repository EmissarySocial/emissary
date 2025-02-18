(function() {

	"use strict";

	var receiveMessage = function(event) {

		var message = event.data
		console.log(message)

		if (message == null) {
			return;
		}

		if (typeof message !== "object") {
			return;
		}

		if (Array.isArray(message)) {
			return;
		}

		if (event.data.action != "resize") {
			return;
		}

		// Find and update the iframe that matches the message src.
		var target = document.querySelector("iframe[src='" + message.src + "']");

		if (target == null) {
			console.log("embedded target not found: " + message.src);
			return;
		}

		target.hidden = false;
		target.scrolling = "no";
		target.style.border = "none";
		target.style.width = "100%";
		target.style.height = event.data.height + "px";
		target.style.overflow = "hidden";
	}

	window.addEventListener("message", receiveMessage)

})();