# Admin Common

Shared base template for every page in the server admin console. It renders nothing on its own — [template.hjson](template.hjson) declares `model: None`, no actions, and only the single `default` state — and exists purely to be inherited by the other `admin-*` templates, each of which lists `extends: ["admin-common"]`.

Inheritance is resolved at load time by `calculateInheritance` in `service/template.go`, which merges the parent's schema, states, actions, and compiled HTML templates into each child (`model/template.go` `Inherit`). That is what lets every admin page invoke the two partials defined here by file name: [menubar.html](menubar.html), the console's top navigation (General / Navigation / Rules / People / Search / External, plus contextual sub-menus keyed off the builder's `.Token`), and [startup.html](startup.html), a banner shown while the domain is still in startup mode that links back to the `/startup` checklist. Child pages include them with `{{template "menubar" .}}` and `{{template "startup" .}}`.

To add a new admin section, create a new `admin-*` folder with `templateRole: admin`, `containedBy: ["admin"]`, and `extends: ["admin-common"]` (both role and containment are enforced by `Template.LoadAdmin` in `service/template.go`), then add its link to [menubar.html](menubar.html) so it appears in the console navigation.
