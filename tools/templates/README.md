# templates

This package builds the `template.FuncMap` that every HTML template in Emissary is parsed with — the helpers available to template designers inside `{{ }}`. It starts from rosetta's [funcmap.All()](https://github.com/benpate/rosetta) and layers on the application's own helpers: date and time formatting (`humanizeTime`, `tinyDate`), color parsing, search highlighting, ActivityStream collection access, and icon rendering.

The layering is also where the application overrides a rosetta helper it does not want. `markdown` is the one that matters: rosetta's version converts without sanitizing, so this package replaces it with [tools/markdown](../markdown/), the single converter whose output is sanitized with the application's policy. Any template writing `{{ .Something | markdown }}` therefore gets safe HTML. `markdownHardWraps` goes through the same converter and sanitizer, and also renders every line break as a `<br>`; it is for a value rendered at view time, where the `process-content` step's `hard-wraps` option does not reach. If you add a helper here that returns `template.HTML`, you are declaring its output trusted and exempt from Go's auto-escaping — sanitize inside the helper, or don't return `template.HTML`.

`customCSS` is the same idea for a stylesheet that a person wrote. It returns `template.CSS` only after [tools/customcss](../customcss/) accepts the stylesheet, and an empty value otherwise, so a template can place it in a `<style>` element. rosetta's own `css` helper is a raw cast; never hand it a stylesheet that someone other than the server's operator wrote.

One consumer builds its funcmap independently: [tools/templatemap](../templatemap/) calls `funcmap.All()` directly, so helpers overridden here are not overridden there.
