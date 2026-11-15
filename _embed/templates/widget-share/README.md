# Share Button

This widget (widgetId `share`) adds a "Share Page" button that invokes the operating system's native share sheet, letting visitors send the current page to any app their device supports. It has no schema or settings form, so there is nothing to configure.

[widget.hjson](widget.hjson) holds only the id, label, and description; [widget.html](widget.html) renders a `div` with `role="button"` that installs theme-global's `Share` behavior. The stream's `.URL`, `.Label`, and `.Summary` travel in `data-share-url`, `data-share-title`, and `data-share-text`, never inside the script, because a quote in a title would otherwise end the hyperscript string (see [../AGENTS.md](../AGENTS.md)). The Web Share API needs a secure context and is missing from some desktop browsers, so the button is sent `hidden` and the behavior shows it only where `navigator.share` exists.
