# Base Intent

Shared base template (category: Base, model: None) that provides the modal for outbound Activity Intents: a visitor wants to like, share, reply to, or follow something here from their own Fediverse account. Templates take its `intent` action by listing `base-intent` in their `extends` array. Embedded consumers are `user-outbox` and `stream-outbox-message`, and the Atlas, Bandwagon, and Qwertylicious packages extend it too.

<<<<<<< Updated upstream
The `intent` action in [template.hjson](template.hjson) opens [intent.html](intent.html) as a modal: it switches its heading on the `?intent=` query parameter (announce, create, dislike, follow, like; a `create` with an `?inReplyTo=` is headed as a reply), remembers previously used Fediverse accounts in browser localStorage via hyperscript, and submits new handles to `/.intents/discover` to locate the account's home server before redirecting. The `follow-email` action offers the non-Fediverse alternative — [follow-email.html](follow-email.html) posts a name/email form to `/.follower/new` — and `follow-email-sent` renders the double-opt-in confirmation page. Note that [follow-email.html](follow-email.html) invokes a `follow-heading` partial it does not define; the extending template must supply it (the embedded `user-outbox` template does).
=======
The `intent` action in [template.hjson](template.hjson) opens [intent.html](intent.html) as a modal. The modal runs on [CamperJS](https://github.com/EmissarySocial/camperjs), in the visitor's browser: it lists the accounts saved on this device, adds new ones, and opens the chosen intent on the visitor's home server.
>>>>>>> Stashed changes

## Additional Information

### Query parameters

The button that opens the modal names the intent and its values in the query string, and the modal copies each into a `data-camper-*` attribute for CamperJS.

| Parameter | Use |
| --- | --- |
| `intent` | `announce`, `create`, `dislike`, `follow`, `like`, or `reply`, in any case. Empty opens the account list alone; anything else shows "Invalid Intent" |
| `object` | The URL of the thing being liked, shared, or followed |
| `inReplyTo` | The post being answered. A `create` that names one is treated as a `reply` |
| `content`, `name`, `summary`, `type` | Values for a `create` |

A `reply` falls back to the visitor's Object intent when their server's Create takes no `{inReplyTo}`, which opens the post on their own server, where a reply threads.

### What the browser needs from the server

Adding an account loads the visitor's actor through `/.proxy`, which answers anonymous visitors for actor documents only (see [handler/AGENTS.md](../../../handler/AGENTS.md)). Nothing else in the modal calls Emissary. The Go handlers this modal used before 2026-10-29, `/.intents/discover` and `/.intents/:intent`, are kept until the CAMPERJS-MIGRATION plan retires them.

Accounts that the older modal saved under the localStorage key `accounts` are not carried over: a returning visitor enters their handle again (CAMPERJS-MIGRATION D23).
