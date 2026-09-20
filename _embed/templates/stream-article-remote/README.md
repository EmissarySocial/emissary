# Remote Article

A Remote Article (`templateId: article-remote`) is a Stream whose body is a Markdown file hosted somewhere else — a README in a Git repository, a gist, a file on a CDN. Emissary fetches that file over plain HTTPS and writes it into the Stream's content, so the page renders with the site's own theme, navigation, search, and URL while the text lives in version control.

It extends [article-base](../stream-article-base/), so it has the same layout, widgets, stylesheet, children, properties, sharing, and publish workflow as any other article. What it does not have is a body editor: `edit` is the screen that says where the body comes from, and `edit-source`, `sync-source`, and `delete-source` each run `with-stream-source` to reach the `StreamSource` record, which lives in its own collection and cannot be reached by `set-data` or `save` on the Stream.

The subsystem behind this Template — the `StreamSource` record, the HTTPS adapter, the synchronization, and the webhook endpoint — is described in `emissary-specs/projects/GIT-MARKDOWN-TO-STREAM-CONTENT.md`.

## Additional Information

### Inheritance is additive, so opting out means overriding

`Template.Inherit` copies a parent action only when the child has not defined one. A child can therefore override an action but can **never remove** it, which is why `editor` and `upload-image` are present here. Each names no roles, which empties its access list so the menu bar hides it, and each forwards to `edit` — because a Domain Owner passes every permission check regardless of the access list, so the empty roles alone would not stop one.

The draft workflow is **not** among the overrides. A remote article takes the same route to the live web as any other: edit, then **Promote**, which is the one control that publishes. `promote-draft` inherits article-base's pipeline whole (`promote-draft`, then `save-and-publish`, `search-index`, `refresh-page`, `forward-to`), and `save-and-publish` is the only thing that moves a Stream inside `withinPublishDate()` (`publishDate < now AND unpublishDate > now`). `Common.Navigation` and `makeStreamQueryBuilder` both AND that filter in, with **no domain-owner bypass**, so an article that never promotes is missing from every auto-generated navigation list — for everyone, including the owner who created it. An earlier revision neutralized `promote-draft` here and reached exactly that dead end. `TestEmbeddedTemplates_MenubarActions` in [service/template_menubar_actions_test.go](../../../service/template_menubar_actions_test.go) pins the pipeline against being overridden away again.

This Template also renders article-base's [edit-menubar.html](../stream-article-base/edit-menubar.html) unchanged, so every tab and button added there appears here too. It once shipped a slim shadowing copy to hide Promote and Discard Draft; both that copy and the reason for it are gone.

**Known defect, not yet fixed:** promoting a draft copies `content` from the draft onto the live Stream, and a draft is a snapshot taken when it was created. On a remote article the body belongs to the synchronization, so a promote can overwrite whatever the last sync fetched with older content — or with nothing at all, if the draft was made before the first sync ever ran. `edit` runs inside `with-draft`, so merely opening the settings screen is enough to create that snapshot. Nothing warns, and the page simply reverts. The sync cannot paper over this and must not try: D4 of [GIT-MARKDOWN-TO-STREAM-CONTENT.md](../../../../emissary-specs/projects/GIT-MARKDOWN-TO-STREAM-CONTENT.md) reserves every state transition to a human.

### Three properties are load-bearing, and each fails silently

**`content.html` is `format: "unsafe-any"`, not the `format: "html"` that article-base declares.** `Schema.Normalize` runs on every save, and rosetta's `html` format re-sanitizes with a plain bluemonday `UGCPolicy` — which allows neither styling nor `iframe`. The value it would be re-sanitizing has *already* been sanitized by the Markdown renderer using Emissary's own policy, which calls `AllowStyling()` and permits embeds. Losing this override deletes syntax highlighting and embedded video from every page, and reports nothing. A child's `String` element replaces the parent's whole, so redeclaring the property is what makes the override stick. The same reasoning is written on the `html` property of [widget-markdown](../widget-markdown/widget.hjson).

**`tagPaths` names `data.tags`, and it is written here rather than inherited.** `Template.Inherit` deliberately does not pass `TagPaths` down, so extending article-base brings none, and `Stream.Save` skips `CalculateTags` entirely when the list is empty — which would leave the Hashtags field in the `properties` form storing a value nothing ever reads. The two go together, and `TestArticleRemoteTemplate_TagsAreEnabled` checks both ends.

One consequence is worth knowing before tagging a documentation page. Tags are extracted only from `data.tags`, but `applyHashtagLinks` then rewrites `content.HTML`, linkifying every literal occurrence of those names in the body. Tag an article `#include` and the `#include` in its C samples becomes a link to a search page. That is the same trade every other article Template makes; what is specific here is that the body arrives from somebody else's file, so the author tagging the article may not be the person who wrote the text it rewrites.

**Nothing edits the body.** The remote file is the source of truth, so an editable body would invite a change that the next synchronization discards without saying so. `properties` edits the Stream's own label, token, summary, icon, and hashtags; front matter in the remote file overrides the first three on the next sync.

[../../../service/template_articleRemote_test.go](../../../service/template_articleRemote_test.go) pins all three against the **resolved** template, because reading this file alone would prove nothing — article-base declares the same properties, so only the merged result says which one won.

### `widgets`, `widget`, and `style` drop article-base's `with-draft` wrapper

A draft reaches the live page only through `promote-draft`, which this Template neutralizes. Left inherited, a layout or stylesheet change would save into a draft, report success, and never appear. Promoting instead is not an option: a draft carries a copy of the body from when it was made, so promoting one would overwrite whatever the last sync fetched.

### Saving the source record IS the request to synchronize

There is no "sync" step and no flag. `StreamSource.Save` writes `LOADING` and queues one signature-deduplicated task, so the **Sync Now** button is a pipeline containing nothing but `{do:"save"}`. A repeat costs one conditional `GET` that most forges answer with a `304`, so the cheap case is genuinely free.

The consequence to know: a caller that saves many `StreamSource` records in a loop fans out one outbound fetch per record, with nothing at the call site to say so. The rule is written on `Save` itself in [../../../service/streamSource.go](../../../service/streamSource.go).

### The settings screen renders from the Stream builder, not from inside `with-stream-source`

`build.Model` — the builder that `with-stream-source` switches to — returns `false` from `UserCan` and `""` from `Permalink` and `BasePath`. A settings page rendered inside the container would therefore have no working action URLs and no permission checks, and every button on it would be dead. [edit.html](edit.html) instead renders from the Stream and reaches the record through the `.StreamSource` accessor on `build.Stream`, which returns an EMPTY record when the Stream has none yet.

That empty record is deliberately not `model.NewStreamSource()`: the constructor mints a webhook token, and a BSON decode leaves alone any field the stored document does not carry, so a load target built that way can hand back a stored record wearing a token nobody installed. See [../../../model/AGENTS.md](../../../model/AGENTS.md).

### Nothing polls, so an uninstalled webhook is an article that silently stops updating

A static file cannot offer a subscription, so the webhook URL on the settings screen has to be pasted into the repository by hand. Until somebody does, the article holds whatever its save-time synchronization fetched, and nothing anywhere reports that. That is why `edit.html` shows `Status` and the time of the last check at the top: a record nobody finished configuring has to read as unconfigured rather than as working.

The token is shared on purpose — give several articles the same token and one webhook refreshes all of them.
