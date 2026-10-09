# build

This package contains the *builders* that Emissary passes to HTML templates when rendering pages. A builder wraps a single model object and exposes a safe, template-friendly view of it — guarding against direct access to protected fields and adding convenience queries for related records (for example, the Stream builder offers `Ancestors`, `Parent`, `Siblings`, and `Children`). This package also implements every action step available to template designers in their pipelines.

See the [project README](../README.md) for the big picture, and [model/step](../model/step/) for the data side of each pipeline step.

## How the builders relate

Every builder starts from `Common`, which carries the request, the database session, the factory, and the visitor's authorization, and exposes the domain-level accessors that every page needs. Three branches grow from it: `CommonWithTemplate` for pages driven by a Template action, `Theme` for the pages a handler renders on its own, and `Registration`, which is action-driven but by a `model.Registration` rather than a `model.Template`.

```
Common                                    request · session · factory · authorization · domain accessors
│
├── CommonWithTemplate                    + model.Template  → pipeline builders (Template actions)
│    ├── Attachment
│    ├── Conversations
│    ├── Domain                           admin settings
│    ├── Follower
│    ├── Group
│    ├── Identity
│    ├── Inbox
│    ├── Model
│    ├── Navigation
│    ├── Notifications
│    ├── Outbox
│    ├── Rule
│    ├── SearchTag
│    ├── Settings
│    ├── Stream
│    │    └── Widget                      sub-builder; embeds *Stream, scopes a pipeline to one widget
│    ├── Syndication
│    ├── User
│    └── Webhook
│
├── Theme                                 handler-rendered theme pages (no Template action)
│    ├── (used directly)                  user-signin · user-signout · guest-signin
│    │                                    reset-password · reset-error · reset-confirm · checkout-claim
│    ├── PasswordReset          + model.User
│    │                                    reset-code · reset-code-invalid · reset-code-inactive
│    └── OAuthAuthorization     + OAuthClient · OAuthAuthorizationRequest · *model.User
│                                         oauth
│
└── Registration                          pipeline-driven, but by a model.Registration rather than a
                                          model.Template — carries its own _action/_actionID.
                                          Its templates supply their own <head>, so it never renders
                                          includes-head and is outside the theme-page contract.
```

The `Theme` branch exists because those pages are served straight from a handler — there is no Template action to build them — yet they still render the Theme's shared partials, so their dot has to offer the same accessors every other page's dot does. `Theme` supplies them, and marks the pages `noindex`; `PasswordReset` and `OAuthAuthorization` embed it and add only what their own templates need.

## Navigation: `redirect-to` vs. `forward-to`

Two steps send a visitor from this page to another URL, and [navigation.go](navigation.go) holds the one decision that separates them. There are two mechanisms on the wire and they are not interchangeable, because they replace different things:

- An **HTTP redirect** (`Location` plus a 3xx) is performed by whatever transport made the request. A browser following a link navigates the whole document, but htmx's XHR follows the redirect invisibly and swaps the response into the current page as a *fragment*.
- The **`Hx-Redirect` header** is performed by htmx itself, which assigns `location.href`, so it always navigates the whole *document*. It means nothing to a browser following a plain `<a href>`, which would land on an empty 200.

So the two steps express two different intents, and each falls back to the other's mechanism in the one case where its own cannot work:

| Step | Intent | htmx caller | Everyone else |
| --- | --- | --- | --- |
| `redirect-to` → `navigateContent` | the content lives at another URL | HTTP redirect, swapped in place — *unless* the target is off-origin, where CORS makes a swap impossible and the navigation is handed to htmx | HTTP redirect |
| `forward-to` → `navigateDocument` | the visitor goes somewhere else | `Hx-Redirect` | HTTP 303, because `Hx-Redirect` is inert here |

`isOffOrigin` asks only whether the target names a host of its own, rather than comparing it to the current request's origin. That is deliberate and fail-safe: treating an absolute *self*-URL as off-origin costs one document navigation where a fragment swap would have done, while a wrong same-origin match — a dropped port, a scheme mismatch — produces a click that silently does nothing at all.

Both steps route their `url` through `navigationURL`, which is also the single place the `uri.IsSafeRedirectURL` guard lives, and `add-stream`'s own `redirect-to:` property goes through the same pair. Neither step writes to the `http.ResponseWriter`: they return a `PipelineBehavior` carrying the status and headers, and `AsHTML` writes the response. Committing a response inside a step would discard the Content-Type and every `HX-Trigger` event earlier steps had accumulated.

## Widget save pipelines

Most builders wrap a top-level record, but [Widget](builder_widget.go) scopes a pipeline to a single Widget embedded in a Stream, so that steps read and write the Widget's own data instead of its container's. Its `object()` is the StreamWidget and its `schema()` nests the Widget definition's schema beneath `data`, which is why steps address Widget properties as `data.{property}` — and why every write is still validated by the Widget's own schema.

That builder exists to run the `saveSteps` pipeline a Widget definition may declare. [executeWidgetSaveSteps](widget_save.go) runs those pipelines for every Widget in the Stream being saved, called from the `save` step before the record is written, so a Widget can derive stored values once at save time instead of recomputing them on every page view. Widgets that declare no pipeline cost nothing, and objects that are not Streams skip the pass entirely. The shipped example is [widget-markdown](../_embed/templates/widget-markdown/), which converts Markdown source into sanitized HTML with one `set-data` step.
