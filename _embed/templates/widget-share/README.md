# Share Button

This widget (widgetId `share`) adds a "Share Page" button that invokes the operating system's native share sheet, letting visitors send the current page to any app their device supports. It has no schema or settings form, so there is nothing to configure.

[widget.hjson](widget.hjson) holds only the id, label, and description; [widget.html](widget.html) renders a `div` with `role="button"` whose hyperscript `script` attribute calls `navigator.share({...})` on click, passing the stream's `.Label` as the title, `.URL` as the url, and `.Summary` as the text, with the share icon supplied by the `icon` template function. Note that the Web Share API requires a secure context and is not available in every desktop browser — the widget does not feature-detect, so the button renders regardless and the click is a no-op where `navigator.share` is undefined; adding a visibility check in the hyperscript would be the natural first extension.
