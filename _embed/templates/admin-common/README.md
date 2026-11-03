# Admin Common

Shared base template for every page in the server admin console. It renders nothing on its own — [template.hjson](template.hjson) declares `model: None`, no actions, and only the single `default` state — and exists purely to be inherited by the other `admin-*` templates, each of which lists `extends: ["admin-common"]`.

Inheritance is resolved at load time by `calculateInheritance` in `service/template.go`, which merges the parent's schema, states, actions, and compiled HTML templates into each child (`model/template.go` `Inherit`). That is what lets every admin page invoke the partials defined here by file name.

- [menubar.html](menubar.html) lays out the page as user-settings does: the primary sections in a left-hand sidebar, and the open section's pages as tabs across the top, chosen by the builder's `.Token`. On mobile the sidebar is left out, and a back arrow opens the full-page menu, `admin-domain`'s `mobile` action ([../admin-domain/mobile.html](../admin-domain/mobile.html)). It also draws [startup.html](startup.html).
- [admin-menu.html](admin-menu.html) lists the primary sections. Its dot is the current Token, a string, and the mobile menu passes `""`.
- [startup.html](startup.html) is a banner shown while the domain is still in startup mode, linking back to the `/startup` checklist.

**menubar.html opens `.app` and `.app-content` and leaves both open.** A page calls `{{template "menubar" .}}` first, with no wrapper of its own, and its own closing `</div>` closes `.app-content`. The browser closes `.app` at the end of `<main>`. Some pages, such as `admin-domain/index.html`, close nothing at all, because later steps of their action (an `edit` form) render below them. Do not wrap a page in another `<div>` before the call, or that `</div>` will close the wrong element.

To add a new admin section, create a new `admin-*` folder with `templateRole: admin`, `containedBy: ["admin"]`, and `extends: ["admin-common"]` (both role and containment are enforced by `Template.LoadAdmin` in `service/template.go`). Then add it to [admin-menu.html](admin-menu.html) as a primary section, or to a section's tabs in [menubar.html](menubar.html).
