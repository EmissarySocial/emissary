# Byline

This widget (widgetId `byline`) displays author attribution and publishing information for a stream: the author's avatar, a link to their profile, and the publish date. It has no schema or settings form, so it renders the same way wherever it is placed.

[widget.hjson](widget.hjson) holds only the id, label, and description; [widget.html](widget.html) reads `.Author` (a `model.PersonLink` from the embedded Stream builder) and conditionally renders the `IconURL` image, the `Name` linked to `ProfileURL` when one exists, and a `<time>` element built from `.PublishDate` using the `isoDate` and `shortDate` template functions from the shared funcmap. The markup carries microformats2 classes (`h-card`, `u-photo`, `p-name`, `u-url`, `dt-published`) so the byline is machine-readable by IndieWeb parsers; keep those classes intact when restyling, and add a schema/form to widget.hjson if you want per-placement options.
