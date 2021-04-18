htmx.defineExtension("authorization", {
    onEvent: function(name, evt) {
        if (name == "htmx:configRequest") {
            var authorization = sessionStorage.getItem("Authorization");
            if (authorization != null) {
                evt.detail.headers["Authorization"] = authorization;
            }
        }
        return true;
    },
    transformResponse: function(text, xhr, _elt) {
        var authorization = xhr.getResponseHeader("Authorization");
        if (authorization != null) {
            sessionStorage.setItem("Authorization", authorization);
        }
        return text;
    }
})