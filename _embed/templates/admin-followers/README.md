# Followers

Admin console page listing the external services that follow this server's search index (the built-in `@search` actor), served at `/admin/followers` under the "Search > Followers" sub-menu. [template.hjson](template.hjson) declares `model: Domain` so `handler/admin.go` builds it with `build.NewDomain`, and all actions are owner-only.

The `index` action renders [index.html](index.html), which shows subscribe-instruction cards for ActivityPub (`@search@hostname`) and RSS (`/@search/feed`) plus a table of current followers from `.Followers.ByName`. The `edit` and `delete` actions both wrap their steps in `with-follower` to load the individual Follower record from the URL's objectID; `edit` then opens a modal whose body [edit.html](edit.html) is currently just a raw JSON dump of the object (`{{.Object | json}}`), a debug-level placeholder rather than a finished form. Extends `admin-common` for the shared menubar partial.
