# Default Notifications: Notes for Agents

The [README](README.md) describes how the notifications pages work. These are the rules that look removable but are not.

## The All tab is `?type=ALL`, never an empty type

The sidebar, menu, and header read `first (.QueryParam "type") "MENTION"`, so an empty type selects Messages & Replies. `notificationTabTypes` maps `ALL` to "every type", and the Mark All Read button on that tab posts `?type=ALL`. The bare `/@me/notifications` URL stays on Messages & Replies (Ben, 2026-11-18).

## One notification is marked read by a POST, never by a GET

A prefetch, a link preview, or a crawler must not clear a notification. `mark-notifications-read` does nothing on a GET, and the reply-controls run checks it (6d). Opening the quote modal is a GET, so the modal marks nothing; the row's click does.

## `read-one` does not fire `refreshPage`

`refreshPage` redraws `main`, which would redraw the list under the modal or the browse view that the same click is opening. The row clears its own dot in the browser: its script removes the `unread` class, and the stylesheet hides the dot of any row without that class. `read`, the Mark All Read action, does fire `refreshPage`.

## The row's script is on the row, and holds no remote value

The `script` attribute in [row.html](row.html) sits on the row's own element, because a listener on a parent would hear every htmx request made inside it. The only value written into it is the notification's own ID, which this server made. Never write an actor name, a URL, or a summary into a `script` attribute, where nothing filters it.

## The mark-read request must outlive the page

A direct-message row is a real navigation to `/@me/conversations`, and a Like or Boost row is one to the User's own post, so the page is gone before a normal request finishes. The row's `fetch` sets `keepalive:true` for that reason. Do not replace it with an htmx request, which is cancelled when the page unloads.

## Values read inside a tag are hoisted into lowercase variables

The HTML minifier runs before html/template parses a file, and it lowercases template code that sits in attribute position inside a tag. `{{if $notification.IsConversation}}` inside the `<a>` tag arrives as `.isconversation` and fails when the row is drawn. [row.html](row.html) reads each such value once, in element content, into an all-lowercase variable. Code inside a quoted attribute value is not rewritten.

## A private message never opens the browse view

`/@me/newsfeed/browse` is the public ActivityStream viewer, so sending a direct message there leaks the conversation into a public context. A DIRECT row links to the Conversations app, the only surface that can decrypt MLS, and it is a plain `href` because that app has its own hash routing.

## The quoting post is another server's HTML

[quote.html](quote.html) prints the post's content only through `htmlMinimal`, as the browse view does, and falls back to the notification's `ObjectSummary`, which is plain text that html/template escapes. A post that the viewer's rules refuse comes back without content, and the modal shows the summary.

## Reject cannot be undone on Mastodon

A rejected quote is revoked with a `Delete` of its `QuoteAuthorization`, and Mastodon does not restore a quote whose authorization comes back (BUG-286 §2). The modal says so before the button, and the button confirms.

## Quotes belong to the Quotes tab only

`notificationTabTypes` no longer adds QUOTE to Messages & Replies (Ben, 2026-11-18). No type is listed by two typed tabs, so the All tab's unread count is the sum of the others; `TestNotificationTabTypes_NoTypeInTwoSections` pins it.

## A Like or a Boost opens the post, and only a Follow opens the actor

A LIKE, DISLIKE, or ANNOUNCE row links to the notification's `ObjectURL`, which is the User's own post that was liked or boosted (Ben, 2026-11-19). A FOLLOW row opens the follower's profile, because a FOLLOW's `ObjectURL` is the User's own actor URL, which tells them nothing.

## A link row needs `role="link"` to highlight

The theme highlights a hovered row of a `div.table` only when it has `role="link"`, `role="button"`, or the `clickable` class (`theme-global/stylesheet/03-widgets-tables.css`). An `<a href>` has none of them by default, so the rows that navigate (direct messages, likes, boosts) set `role="link"` themselves. Without it they still work and never highlight.

## An avatar that fails to load is replaced by the placeholder

`Actor.IconURL` is a snapshot taken when the notification arrived, so it can point at an image the other server has since moved or deleted. The image has an empty `alt`, so a failed one draws nothing at all. Its `onerror` hides it and shows the placeholder that follows. The placeholder sits in a `<span hidden>` because its own `flex-center` class would override the `hidden` attribute.
