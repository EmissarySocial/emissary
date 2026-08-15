# Syndication

Admin console page for managing the syndication targets users can cross-post their content to, served at `/admin/syndication` under the "External > Syndication" sub-menu. [template.hjson](template.hjson) declares `model: Syndication`, the one admin template built with `build.NewSyndication` in `handler/admin.go`; both actions are owner-only.

The `index` action renders [index.html](index.html) (plus an `Hx-Push-Url` header via `set-header`), which embeds the `table` action inline through `{{- .View "table" -}}`. That `table` action has no HTML file: it is an `edit-table` step over the `syndication` path with a `layout-table` form (token, label, description, URL per row) followed by `save`, giving an in-place editable rows UI generated entirely from the hjson. Extends `admin-common` for the shared menubar partial.
