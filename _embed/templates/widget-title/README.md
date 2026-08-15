# Title Block

This widget (widgetId `title`) displays the stream's title as a semantically meaningful `<h1>`, followed by its summary text when one exists. Placing it in a layout gives the page a proper machine-readable heading instead of leaving the title to ad-hoc markup.

[widget.hjson](widget.hjson) holds only the id, label, and description — there is no schema or form, so the widget has no settings. [widget.html](widget.html) renders `.Label` from the embedded Stream builder inside `<h1 class="p-name">` and, when `.Summary` is non-empty, a `<div class="p-summary">` beneath it; the `p-name` and `p-summary` classes are microformats2 properties that IndieWeb parsers read, so keep them when restyling.
