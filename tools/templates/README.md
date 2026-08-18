# templates

This package builds the `template.FuncMap` that every HTML template in Emissary is parsed with — the helpers available to template designers inside `{{ }}`. It starts from rosetta's [funcmap.All()](https://github.com/benpate/rosetta) and layers on the application's own helpers: date and time formatting (`humanizeTime`, `tinyDate`), color parsing, search highlighting, ActivityStream collection access, and icon rendering.

The layering is also where the application overrides a rosetta helper it does not want. `markdown` is the one that matters: rosetta's version converts without sanitizing, so this package replaces it with [tools/markdown](../markdown/), the single converter whose output is sanitized with the application's policy. Any template writing `{{ .Something | markdown }}` therefore gets safe HTML. If you add a helper here that returns `template.HTML`, you are declaring its output trusted and exempt from Go's auto-escaping — sanitize inside the helper, or don't return `template.HTML`.

One consumer builds its funcmap independently: [tools/templatemap](../templatemap/) calls `funcmap.All()` directly, so helpers overridden here are not overridden there.
