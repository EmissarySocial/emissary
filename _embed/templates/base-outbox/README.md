# Base Outbox

Shared base template (category: Base, model: Outbox, no `templateRole`) that supplies what every outbox shares: the Profile tabs of the settings area, and the Follow modal with the email pages it leads to. Every outbox template lists `base-outbox` in its `extends`, so each outbox decides what its users see under Settings → Profile and when a visitor follows them. Embedded consumers are `user-outbox`, and the external Atlas, Bandwagon, and Qwerty outboxes extend it too. It declares the `Outbox` model so that its forms validate against the User schema, and it has no `templateRole`, so no user can choose it as a profile template.

## How the tabs load

The settings page at `/@me/settings/general` ([../user-settings/general.html](../user-settings/general.html)) draws the settings sidebar and a `#profile-settings` area, and renders the General tab into that area on the server through the Settings builder's `.OutboxView "settings-general"`, which runs the action from the user's own outbox template. So the page arrives complete, and the navigation's view transition goes straight to real content. Each tab is its own outbox action, and clicking one swaps the area for that action with `hx-push-url="false"`, so the address stays `/@me/settings/general` and a reload returns to General. The swap inherits the page's `hx-swap="innerHTML transition:true"`, so moving between tabs runs a view transition too.

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

A tab link is an `hx-get` to `/@me/settings-<tab>` with `hx-target="#profile-settings"` and `hx-push-url="false"`, never a boosted link, and it sets no `hx-swap`, so it keeps the page's view transition. Each tab renders inside the settings page, so it uses the settings page's markup (`#menu-bar`, then a `card` with `transition-main-card`) rather than the outbox theme's.

Import and Export are not Profile tabs. They are a Migration section of their own in [../user-settings](../user-settings/README.md), so no outbox can remove them.

## The Follow modal

The default profile's Follow button and the follow widget open `/@{user}/follow`, which draws [follow.html](follow.html). It lists the visitor's saved Fediverse account through CamperJS, then a button for each way to follow:

- **Fediverse,** [follow-activitypub.html](follow-activitypub.html), where the visitor adds their account by its handle. It shows only while no account is saved, because a visitor keeps one at a time.
- **Bluesky,** a link to the profile's bridged account, shown only when the Bluesky bridge is on.
- **Email,** [follow-email.html](follow-email.html), a name and address form that posts to `/.follower/new`.

[follow-heading.html](follow-heading.html) heads each view. [follow-styles.html](follow-styles.html) draws the account rows, and every view that lists accounts includes it, because each view replaces the modal and takes the last one's styles with it.

Three email pages are reached from URLs that Go builds, never from a link in these views:

| Action | Reached from |
| --- | --- |
| `follow-email-sent` | The redirect after `/.follower/new` saves the follower ([handler/follow_email.go](../../../handler/follow_email.go)) |
| `follower-confirm` | The link in the confirmation email ([email-follower-confirmation/body.html](../email-follower-confirmation/body.html)) |
| `follower-unsubscribe` | The unsubscribe link and `List-Unsubscribe` header of every email a follower receives (`Follower.UnsubscribeLink` in [model/follower.go](../../../model/follower.go)) |

[follow-button.html](follow-button.html) is a Follow button for an outbox to draw with `{{template "follow-button" .}}`. It holds two copies that look the same. One opens the modal. The other follows in one click as the visitor's saved account, with a "..." button beside it that opens `base-intent`'s account manager. CamperJS shows the one-click copy only when the default account can follow (`camper-show-if-ready`), and the server sends it hidden, so the modal's copy works without CamperJS. Qwertylicious's profile card draws it, and the default sidebar does not yet.

An outbox restyles any view by adding one with the same name, as Atlas does for `follow.html`, `follow-activitypub.html`, and `follow-heading.html`. A `follow` action of its own replaces only that action, and the email pages stay.
