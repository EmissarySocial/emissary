
function signIn() {
	window.location = "/"
}

function signOut() {

}


htmx.defineExtension('modal', {

	// _="on closeModal add .closing then wait 150ms then remove me"

	onEvent: function(name, event) {

		var modal = htmx.find("#modal")
		if (name == "closeModal") {

			htmx.addClass(modal, "closing")
			setTimeout(function() {
				htmx.remove(modal)

				if (event.detail && event.detail.nextPage) {
					if (window.location == event.detail.nextPage) {
						window.location.reload()
					} else {
						window.location = event.detail.nextPage
					}
				}

			}, 150)
		}
	}
});
