_hyperscript.config.conversions["FormEncoded"] = function(object) {
    var result = [];
    for (key in object)  {
        var encodedKey = encodeURIComponent(key);
        var encodedValue = encodeURIComponent(object[key]);
        result.push(encodedKey + "=" + encodedValue);
    }

    return result.join("&");
};

htmx.config.useTemplateFragments = true;