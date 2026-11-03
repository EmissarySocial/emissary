/**
 * PRISM: syntax highlighting for code blocks, in themes that load
 * resources/prism-1.30.0/prism.min.js before this bundle.
 *
 * Prism highlights the whole page once on load.  htmx swaps <main> on every
 * navigation, so each swapped element is highlighted again on htmx:load, or
 * every page after the first stays uncolored.  A theme that does not load
 * Prism skips all of this.
 *
 * HJSON is not a Prism component, so it is defined here as JSON that also
 * allows # and // comments and unquoted keys.
 */
(function () {

	// Themes that do not load Prism get nothing from this file
	if (!window.Prism) {
		return
	}

	// HJSON: JSON plus comments and unquoted keys.  Defined before Prism's own
	// first pass, which waits for DOMContentLoaded.
	window.Prism.languages.hjson = window.Prism.languages.extend("json", {
		"property": {
			pattern: /(^|[\s,{])(?:"(?:\\.|[^\\"\r\n])*"|[^\s,:{}\[\]"'#\/]+)(?=\s*:)/m,
			lookbehind: true,
			greedy: true,
		},
		"comment": {
			pattern: /(?:\/\/|#).*|\/\*[\s\S]*?(?:\*\/|$)/,
			greedy: true,
		},
	})

	// Highlight whatever htmx just put on the page
	document.addEventListener("htmx:load", function (event) {
		window.Prism.highlightAllUnder(event.detail.elt)
	})
})();
