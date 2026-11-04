# Default Inbox: Notes for Agents

The [README](README.md) describes how the inbox works. These are the rules that look removable but are not.

## A remote value goes only into an `href`

A remote actor's profile metadata, such as `$attachmentValue` in [browse-actor.html](browse-actor.html), renders only as an `href`, where html/template's URL filter blocks `javascript:` and `data:` links. Never put a remote value into hyperscript or any `script` attribute, where nothing filters it.

## Folder menu links spell out their own htmx attributes

Every link in [newsfeed-menu.html](newsfeed-menu.html) sets `hx-target`, `hx-swap` and `hx-push-url` itself. The menu sits inside the sidebar's own `hx-target` and the sort form's `hx-swap="none"`, which the links would otherwise inherit, and changing folders would stop swapping `<main>` with its view transition.

## The welcome is decided once, in index.html

The one-row `.Inbox.Top1` query in [index.html](index.html) is the only test for an empty, unfiltered inbox or folder. When it is empty, index.html passes `?welcome=inbox` or `?welcome=folder` to the list, and [list.html](list.html) and [list-social.html](list-social.html) render [newsfeed-welcome-inbox.html](newsfeed-welcome-inbox.html) or [newsfeed-welcome-folder.html](newsfeed-welcome-folder.html). The welcome carries its own heading and Follow Someone button, so index.html drops its own: the button always, and the heading on desktop only, because on mobile the heading holds the back link to the folder menu. The welcome's follow link omits the folderId, so the follow modal defaults to the first folder.

## Hidden posts reveal through `?reveal=true`

A post hidden by the viewer's rules renders as a placeholder. Inside a reply tree, clicking it re-renders the same view with `?reveal=true`, which lifts only that post's hide. In the list, the placeholder opens the `browse` modal instead, which shows its main post without a rule check, so only that post is revealed and each reply below stays gated on its own rules.

## Every way into a post opens `browse`

A list row, the "N Replies" link, and a hidden post's placeholder all open the `browse` modal. Do not add a second view of a post; extend `browse`. The inbox kept one until BUG-331, the `message` modal, and nobody noticed when it broke: its replies failed from August 2025, its Code button never worked, and its Prev and Next went nowhere after the `newsItemId` rename. `TestBuilderDots_UserInboxLinks` in [build/builderDots_test.go](../../../build/builderDots_test.go) fails when any link in this folder names an action that does not exist.

## The note form posts to itself

`annotation-editor` posts to its own `endpoint`, not to the modal, because an `as-modal` POST always closes the modal. After a save it renders [browse-annotation-modal.html](browse-annotation-modal.html), which redraws the whole modal so the Notes / Noted label is current.

## The Inbox redraws the browse modal, never `with-annotation`

The `view-html` that renders `browse-annotation-modal` sits after `with-annotation` closes, in both `annotation-editor` and `annotation-delete`. Inside `with-annotation` the dot is the Annotation's `Model` builder, and `.View "browse-annotation"` would render the whole Notes tab with it. That tab reaches `reply-button`, whose `CanReplyTo` only the Inbox has, so the save answered 500 after writing the note (BUG-330). In `annotation-delete` the redraw also waits on `?url=`, because only a document note's Delete link (from `annotation-editor`) carries it; an actor note's Delete closes its modal instead.

## `remove-event` shares a pipeline with `delete`

In `annotation-delete`, the `delete` step adds a `closeModal` event, and the `remove-event` that follows strips it so the browse modal stays open to be redrawn. Keep the two in the same branch: a `remove-event` anywhere else has no `closeModal` to remove.
