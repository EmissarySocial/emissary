htmx.defineExtension('authentication-header', {
    onEvent: function(name, evt) {
        if (name == "htmx:configureRequest") {
            var authentication = sessionStorage.getItem("Authentication")
            if (authentication != null) {
                evt.detaul.headers["Authentication"] = authentication
            }
        }
    },
    transformResponse: function(text, xhr, elt) {
        var authentication = xhr.getResponseHeader("Authentication")
        if (authentication != null) {
            sessionStorage.setItem("Authentication", authentication)
        }
    }
})