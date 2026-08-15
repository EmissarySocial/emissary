# Following

Admin console page listing the external sources this server's search index follows for search results, served at `/admin/following` under the "Search > Sources" sub-menu. [template.hjson](template.hjson) declares `model: Domain` (built with `build.NewDomain` in `handler/admin.go`), and both actions are owner-only.

The `index` action renders [index.html](index.html): a table of `.Following` records showing label, URL, delivery method, and last-polled date, with an "Follow a New Source" row at the top; ActivityPub sources push in real time while other feed types are polled daily. The `create` action wraps `with-follower` around an `as-modal` + `edit` step whose form is defined inline in the hjson — a single `url` field accepting an actor handle like `@search@servername.social` — so there is no create HTML file. Extends `admin-common` for the shared menubar partial.
