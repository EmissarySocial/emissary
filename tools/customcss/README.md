# customcss

This package checks a stylesheet that one person writes for a page that other people view, such as a profile's custom CSS. Unlike rosetta's `css` format, which rewrites a stylesheet down to an allowlist of properties, it never changes the text: `Validate` accepts the author's CSS as written, or returns a validation error naming the rule it breaks and the line, such as `The stylesheet cannot load files from other websites (line 4).`

An accepted stylesheet is safe to place inside a `<style>` element:

- **It cannot close the element.** `"</"` is refused anywhere, including inside strings and comments, so a `data:` SVG must be percent-encoded (`%3C/svg%3E`).
- **It cannot reach past a wrapper.** Every `{`, `(`, and `[` it opens must be closed by its own match, and it cannot end with a lone backslash, which would escape the first character written after it. A template can therefore wrap it in `@scope (…) { … }` and know that it stays inside.
- **It loads nothing from another website.** `@import` is refused, and every address in a `url()`, `image-set()`, `image()`, `src()`, or `cross-fade()` must be relative, a `#fragment`, or a `data:` URI.

The checks run over the tokens of tdewolff's CSS lexer, so a brace or `url()` inside a string or comment counts the way a browser counts it, and escape codes are decoded before any name or address is compared.

Two callers apply the same rules. The package registers the `custom-css` schema format, so a Template's schema checks a stylesheet when it is saved, and `custom-css=remote` relaxes only the rule against other websites. The template function `customCSS` in [tools/templates](../templates/) checks it again when it is rendered, and writes nothing for a stylesheet that fails.
