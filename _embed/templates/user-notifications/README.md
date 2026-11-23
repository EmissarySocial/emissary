# Default Notifications

The signed-in user's notifications section at `/@me/notifications`, split into six tabs: All, Messages & Replies, Quotes, Boosts, Likes, and Followers. [template.hjson](template.hjson) declares `templateRole: user-notifications` on the `Notifications` model (which operates on the User record), and the template is loaded directly by templateId in `build/builder_user_notifications.go`. Unlike inbox and outbox templates, it is not user-selectable. All actions require `roles: ["self"]`, and the template extends `base-social`. [AGENTS.md](AGENTS.md) holds the rules that look removable and are not.

## Tabs

Each tab has a page action and a list action. The page action sets the tab's `?type=` with `set-query-param`, so the shared [sidebar.html](sidebar.html) and [header.html](header.html) and the `.Notifications` query all read the same tab. The page lazy-loads its `-list` action, which also serves the infinite scroll, and opens an SSE connection to `/@me/sse/notifications` that re-fetches the header and the list when a notification arrives.

| Tab | Actions | `?type=` | Lists |
| --- | --- | --- | --- |
| All | `all`, `all-list` | `ALL` | every type |
| Messages & Replies | `replies`, `replies-list` | `MENTION` | MENTION, REPLY, DIRECT |
| Quotes | `quotes`, `quotes-list` | `QUOTE` | QUOTE |
| Boosts | `shares`, `shares-list` | `ANNOUNCE` | ANNOUNCE |
| Likes | `likes`, `likes-list` | `LIKE` | LIKE, DISLIKE |
| Followers | `followers`, `followers-list` | `FOLLOW` | FOLLOW |

The bare URL (`index`) draws Messages & Replies. `notificationTabTypes` in the builder turns a `?type=` into the types listed.

## Rows

Every list draws its rows with one partial, [row.html](row.html), which switches on the notification's type. A click does two things: it marks that notification read, and it opens something.

| Type | Opens |
| --- | --- |
| FOLLOW | the new follower's profile, in the browse view |
| LIKE, DISLIKE, ANNOUNCE | the User's own post that was liked or boosted, by a real navigation |
| MENTION, REPLY | the remote post, in the browse view |
| DIRECT | the Conversations app, by a real navigation |
| QUOTE | the quote modal |

A row whose actor the viewer's rules hide shows label chips in place of the actor. A hidden reaction opens nothing; any other hidden row opens as usual, with `reveal=true` on the browse view.

## Marking read

- **One row:** an unread row posts to `read-one?notificationId=` when it is clicked, clears its own dot, and then fires `refreshSidebar` so the counts follow.
- **A whole tab:** the header's "Mark All Read" button posts to `read?type=`, which fires `refreshSidebar` and `refreshPage`.

Both run the `mark-notifications-read` step. The `badge` action renders [badge.html](badge.html), the red dot that the theme's navigation bar swaps in over SSE and on `refreshSidebar`.

## Quotes

A QUOTE notification announces a quote of the User's post that Emissary approved automatically (FEP-044f). Its row opens the `quote` action, a modal drawn by [quote.html](quote.html): who quoted, the quoting post, and two buttons, Reject Quote and Close.

Reject posts to `reject-quote?collectionItemId=` with the ID of the CollectionItem that approves the quote. The builder's `QuoteItemID` finds that ID from the notification's Stream and quote post, and the button is left out when the quote is no longer approved. The action loads the item with `with-collection-item` and deletes it with the standard `delete` step, which revokes the quote, deletes the notification, closes the modal, and refreshes the page. It names no notification, so any other page can post to it after the notification has been purged.

## Other files

[header.html](header.html) draws no title, because the sidebar already names the tab. On desktop it holds only the "Mark All Read" button, and draws nothing when the tab has nothing unread. On mobile the sidebar is hidden, so the header draws the six tabs as well. Styles live in [stylesheet/](stylesheet/), served as `/.templates/user-notifications/stylesheet`.
