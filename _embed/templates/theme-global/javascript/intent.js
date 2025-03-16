(function(){

	// Displays intent markers based on localStorage
	var repaintIntents = function() {

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

	// Execute repaintIntents() on page load and when the localStorage changes
	window.addEventListener("storage", repaintIntents);
	repaintIntents();

	// Listen for account sign-in events. If empty, add the user's account info to localStorage
	window.addEventListener("signin-account", function(event) {
		console.log("Account signed in");
		console.log(event);
		console.log(event.detail);

		var existingAccounts = JSON.parse(window.localStorage.getItem("accounts"));

		if (existingAccounts != null) {
			if (existingAccounts.length > 0) {
				return;
			}
		}

		account = event.detail;
		account["elt"] = null;

		var accountsToStore = []
		accountsToStore.push(account);
		console.log(accountsToStore);

		var jsonToStore = JSON.stringify(accountsToStore);
		console.log(jsonToStore);
		window.localStorage.setItem("accounts", jsonToStore);
		console.log(window.localStorage.getItem("accounts"));
	})
	
})();