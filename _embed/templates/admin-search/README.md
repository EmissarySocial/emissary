# Search

Admin console page for search-index maintenance, served at `/admin/search` under the "Search > Indexes" sub-menu. [template.hjson](template.hjson) declares `model: Search`, which `handler/admin.go` builds with the same `build.NewDomain` builder used by the other Domain-scoped admin pages; the single `index` action is owner-only and just renders [index.html](index.html).

The page itself is three buttons that POST outside the template pipeline to dedicated owner-only routes registered in `server.go`: `/admin/index-all-streams`, `/admin/index-all-users`, and `/admin/reindex-activitystream-cache`, each with an htmx spinner while the reindex runs. So this template is only the UI shell — extending the reindex behavior means editing those handlers, not this folder. Extends `admin-common` for the shared menubar partial.
