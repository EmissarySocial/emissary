(function(){

	var update = function() {
		// Find signed-in sessions
		var accounts = JSON.parse(window.localStorage.getItem("accounts"));

		if (accounts == null) {
			accounts = [];
		}

		// Label account markers
		var accountMarkers = document.querySelectorAll(".intentMarker-account")
		accountMarkers.forEach(function(/** @type {HTMLElement} */ node) {

			switch (accounts.length)  {
				case 0:
					node.innerHTML = "<span class=text-underline>With Fediverse Account</span>";
					return;

				case 1:
					node.innerHTML = "As " + accounts[0].preferredUsername +" (<u>Change</u>)";
					break;

				case 2:
					node.innerHTML = "As <u>" + accounts[0].preferredUsername + "</u> or 1 other";
					break;

				default:
					node.innerHTML = "As <u>" + accounts[0].preferredUsername + "</u> or " + (accounts.length - 1) + " others";
					break;
			}
		})
	}

	window.addEventListener("storage", update);
	update();

})();