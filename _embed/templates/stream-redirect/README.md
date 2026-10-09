# Redirect

A Redirect (`templateId: redirect`) is a placeholder stream that forwards visitors to another web address stored in its `data.url` schema property. It can be placed at the site's top level or inside a folder or article (`containedBy: ["top", "folder", "article"]`), has `default` and `published` states, and uses the usual `viewer`/`editor` roles — useful for putting an external link into site navigation or a folder listing.

The forwarding is the whole of the `view` action in [template.hjson](template.hjson): a single `redirect-to` step pointed at `data.url`. `edit` renders [edit.html](edit.html), a settings page whose form (URL, label, token, icon, summary) is defined by the `edit-form` action; `create` saves immediately and forwards to that editor so the owner can configure the new stream; `delete` returns to `/admin/navigation`; and visibility is controlled through `sharing` (simple-sharing on the `viewer` role) rather than a publish workflow.

## Additional Information

### Why one step is enough

A Redirect's whole job is a cross-origin hop, and the two ways of performing one are not interchangeable. A plain `<a href>` — the site header's links, or any non-browser client — needs a real HTTP redirect and follows the 307 natively. An `hx-get` cannot: htmx issues an XHR, the XHR follows the 307 itself, and the browser blocks the resulting cross-origin GET under CORS, so the click silently does nothing. htmx instead needs the `Hx-Redirect` response header, which makes *it* assign `location.href` — and that header is meaningless to a browser following a bare link, which would land on an empty 200.

Navigation links carry **both** attributes ([widget-navigation-list](../widget-navigation-list/) renders `<a href="/{{token}}" hx-get="/{{token}}">`), so whichever path a click takes has to work. That is not this Template's problem to solve, though: `redirect-to` detects an off-origin target on its own and hands the navigation to htmx when htmx is the one asking. See [build/navigation.go](../../../build/navigation.go) and [build/README.md](../../../build/README.md#navigation-redirect-to-vs-forward-to) for the decision itself, and [service/template_redirect_test.go](../../../service/template_redirect_test.go) for the tests that keep this action from growing a branch of its own back.

### The `published` state gates anonymous access

`view` grants `["author", "editor"]` unconditionally, and adds `anonymous` only through `stateRoles` on the `published` state. A Redirect left in `default` therefore forwards its owner but not the public, and `edit-form` ends in `save-and-publish` so configuring one through the editor publishes it as a side effect.
