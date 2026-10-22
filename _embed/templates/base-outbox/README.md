# Base Outbox

Shared base template (category: Base, model: Outbox, no `templateRole`) that supplies the Profile tabs of the settings area. Every outbox template lists `base-outbox` in its `extends`, so each outbox decides what its users see under Settings → Profile. Embedded consumers are `user-outbox`, and the external Atlas, Bandwagon, and Qwerty outboxes extend it too. It declares the `Outbox` model so that its forms validate against the User schema, and it has no `templateRole`, so no user can choose it as a profile template.

## How the tabs load

The settings page at `/@me/settings/general` ([../user-settings/general.html](../user-settings/general.html)) draws only the settings sidebar and an empty `#profile-settings` area. That area loads `/@me/settings-general` from the user's own outbox with htmx. Each tab is its own outbox action, and clicking one swaps the area for that action with `hx-push-url="false"`, so the address stays `/@me/settings/general` and a reload returns to General.

Each tab is a pair of actions:

- **A page,** `settings-<tab>`, which renders [settings-tabs.html](settings-tabs.html) and then a card holding the tab's form through `{{.View "settings-<tab>-form"}}`.
- **A form,** `settings-<tab>-form`, which edits and saves the User. Its `endpoint:` option posts back to `/@me/settings-<tab>-form`, so a save returns only the form.

The stock tabs are General ([settings-general.html](settings-general.html)), Images ([settings-images.html](settings-images.html)), and Links ([settings-links.html](settings-links.html)). `settings-icon-delete` and `settings-image-delete` remove the avatar and banner from the Images form. The General and Images forms, and both deletes, end with `search-index`, so a profile edit reaches search.

## Customizing the tabs in an outbox

Inheritance works by name, so an outbox overrides exactly what it defines:

- **One form:** define `settings-<tab>-form`. The stock tab bar and card stay.
- **One tab:** define the page action `settings-<tab>` and its HTML file.
- **The tab bar:** add a `settings-tabs.html` file. Every inherited tab then draws it. It renders against the Outbox builder, so `.IsAction` marks the selected tab and `.IsDesktop` picks a layout.
- **A new tab:** add a `settings-<tab>` page, a `settings-<tab>-form` action, and a link to it in your own `settings-tabs.html`.

A tab link is an `hx-get` to `/@me/settings-<tab>` with `hx-target="#profile-settings"` and `hx-push-url="false"`, never a boosted link. Each tab renders inside the settings page, so it uses the settings page's markup (`#menu-bar`, then a `card` with `transition-main-card`) rather than the outbox theme's.

Import and Export are not Profile tabs. They are a Migration section of their own in [../user-settings](../user-settings/README.md), so no outbox can remove them.
