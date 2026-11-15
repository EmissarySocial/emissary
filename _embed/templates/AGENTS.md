# _embed/templates — Notes for AI Agents

See [build/README.md](../../build/README.md) for how templates and action pipelines fit together, and the `html-templates` skill for comment and formatting conventions. These are the foot-guns — nearly all of them fail **silently**, which is why they are written down.

## The minifier runs before the template parser, and it does not know Go template syntax

`service.loadHTMLTemplateFromFilesystem` minifies with `tdewolff/minify` and only then calls `html/template.Parse`. Any action inside a tag's attribute region, or containing a double-quoted string inside an attribute value, is rewritten:

- `<input value="X"{{if eq $w "MEDIUM"}} checked{{end}}>` becomes `eq $w "medium"` — parses fine, never matches.
- `{{if $isMedium}}` becomes `{{if $ismedium}}` — "undefined variable" at startup, so at least this one is loud.
- `value="{{.QueryParam "username"}}"` becomes `.QueryParam " username"` — the space moves *inside* the string literal.
- An action inside an element that admits only specific children (`<select>`, which admits only `<option>`) is read as a stray text node and **deleted**, taking the whole conditional with it. Repeat the entire `<select>` per branch.

**Put the action outside the tag, and use backticks for string arguments inside an attribute value.** The codebase's `{{.Data \`width\`}}` idiom is exactly this workaround. None of these raise an error — the page just comes out wrong. It was found live in `theme-global/user-signin.html`, where the username prefill had silently never worked. The loader also falls back to unminified source when minification errors, so a `{{...}}` quoted inside an HTML comment as documentation is its own trap.

## htmx `hx-swap` is INHERITED, so `hx-swap="none"` on a container poisons every descendant

`hx-swap`, `hx-push-url`, and `hx-target` all inherit down the DOM. A drag-sort form (`<form class="sortable" hx-post=".../sort" hx-swap="none">`) makes every `hx-boost`/`hx-get` link inside it fire, return 200, and discard the response — htmx's `swap("none")` is a no-op. The click does nothing, with no console error and no server error, and the server *did* receive the request.

**When a boosted link does nothing, check its ANCESTOR chain for an inherited `hx-swap="none"` before suspecting JavaScript.** Files containing both `hx-boost` and `hx-swap="none"` are suspects, but verify the `none` is an ancestor and not a sibling button — a "Mark All Read" button legitimately uses it. To keep boost inside such a form, add `hx-disinherit="hx-swap hx-push-url"` to the form; its own sort POST is unaffected, because self-lookup ignores its own `hx-disinherit`.

## htmx reads `elt.name`, so a form-associated custom element is silently dropped

htmx does not use `FormData`. `processInputValue` iterates `form.elements` and rejects anything whose `elt.name` is empty or null. `form.elements` does contain form-associated custom elements (`static formAssociated = true`), but a custom element has no `name` IDL property unless its class defines one — so its value never reaches the server, with no error anywhere. Put the `name` on a real input inside the element, or define a `name` property on the class.

## An unset hyperscript `:var` is `undefined`, and `is nil` only compares against `null`

In hyperscript 0.9.93, on a fresh element `:foo is nil` is FALSE and `:foo is not nil` is TRUE — the opposite of what you would expect. **Use `exists` / `does not exist` for presence guards.** This silently broke `saveButton._hs`, whose `if :message is not nil then exit end` init-guard fired on every fresh button: width was never set, `.success` was never removed, and the button stuck green. The `no` operator is the other safe choice; it covers null, undefined, and empty at once. A value read from `getAttribute` is a real `null`, so `is null` is correct there.

## Legacy resource folders still ship, and a stale reference fails silently

`theme-global/resources/` carries both the current versioned folders (`hyperscript-0.9.93/`, `htmx-1.9.12/`, `sortable-1.15.7/`, `camperjs-0.2.0/`) and unversioned or older legacy ones. A theme referencing a legacy path gets the old engine, which cannot parse the shared behaviors bundle — bare `focus` is 0.9.93+ — so the whole bundle dies with "Expected 'end' but found 'focus'" and every behavior (Modal, tabs, menu, multiselect) stops installing. The legacy folders are kept deliberately, because **database-stored themes cannot be scanned** and a missing folder 404s into `Sortable is not defined`. Check which folder a theme actually references before debugging a dead behavior.

**A bundle whose markup contract changes needs a new folder, not a new file in the old one.** `handler/resources.go` serves every resource with `Cache-Control: public, max-age=2592000` and no `ETag`, so a browser reuses its copy for 30 days without asking, even across a server restart. `camperjs/` holds CamperJS from before its `data-camper-*` rename and `camperjs-0.2.0/` holds the current one; a browser that kept the old bundle while the templates changed filled every intent placeholder empty, so Follow opened `/@me/intent/follow?object=`. `TestEmbeddedTemplates_LoadVersionedCamperJS` fails if an embedded template names `resources/camperjs/` again.

## Third-party bundles are not self-contained just because they are vendored

EasyMDE 2.21.0 fetches FontAwesome from a CDN on construction unless `autoDownloadFontAwesome: false` is set. Assume any vendored editor or widget issues its own runtime requests, and check for them rather than assuming the bundle is the whole dependency.

## EditorJS's color picker is what keeps a pasted line break

EditorJS 2.31 saves a block with the block tool's own rules, so a typed Shift+Enter `<br>` is kept. It cleans a paste with only the inline tools' rules (`baseSanitizeConfig`), so every pasted `<br>` was dropped and the lines ran together (BUG-326). `stream-article-editorjs/editor.html` therefore registers the color picker as `ColorPickerKeepingBreaks`, which adds `br` to that tool's rule and so to every block that allows inline tools. Removing the color picker, or registering the plain `ColorPicker.default`, brings the bug back without any error. qwertylicious's `qwerty-article` editor carries the same subclass, and its `tests/playwright/editorjs.spec.mjs` pastes into both editors. The scripts load at `@latest` (BUG-264), so a future EditorJS release can change this without any change here.

## `turboclick` on a CONTAINER is intentional

`theme-global/javascript/turboclick.js` fires a synthetic `click()` on mousedown against **the element actually pressed**, not the nearest `.turboclick` ancestor, so nested links, buttons, `hx-get`, and hyperscript `on click` all fire as they would naturally. `class="turboclick"` on a container (modal header, sort-form rows) is therefore safe — do not "fix" it by moving the class onto children. Two guards must stay: the `.folder-handle` bail, without which pressing a SortableJS drag handle navigates instead of dragging (it was dropped once in a cleanup commit while the doc comment still promised it), and the left-button-only, no-modifier test that preserves open-in-new-tab. If a click does nothing inside a turboclick region, the synthetic-click targeting is not the likely cause; look at the htmx swap rule above first.

## A struct dot reaching `includes-head` must implement `IsIndexable()`

`includes-head` evaluates `.IsIndexable`. `html/template` treats a missing field on a **struct** dot as a hard execution error (HTTP 500), while a missing key on a **map** dot silently evaluates nil. Any new lightweight builder that is a struct, does not embed `build.Common`, and renders through `includes-head` must implement `IsIndexable() bool`. `build.OAuthAuthorization` 500'd every `GET /oauth/authorize` this way, the second builder to hit it after `model.Domain`.

## A Template's create seed and its schema default are linked by nothing but a test

`data.width` is `["FULL","LARGE","MEDIUM","SMALL"]` defaulting to `FULL`. Adding `FULL` above `LARGE` silently redefined `LARGE` from 100% to 75%, so four article templates that seeded `LARGE` on create started producing 75%-wide articles. The seed and the default live in different files and nothing connects them — `TestLayoutControls_CreateSeedsMatchWidthDefault` is that connection, so extend the test's reach rather than working around it.

## An `html-remote` element and its `edit-attachments` step are linked by nothing but a test

A multi-file field is an `html-remote` element, the form library's plain loader, whose `url` names the action that serves it, and the `edit-attachments` step on that action draws the whole zone and owns the category, the field name, the types, and the limits. The template writes the URL (Ben, 2026-11-04): `/{{.ID}}/photos` on an edit form, and the parent's add with the action in its query on a create form, `/{{.ParentID.Hex}}/add?action=photos`, because the object has no route of its own yet; a chooser-style add needs `&templateId={{.TemplateID}}` too, and a top-level template writes its own add route. So a mismatch is a template that loads and then fails in the browser: a URL naming no such action on the template, or one without `edit-attachments`, draws a shell that never fills. A create form's `create` action must run that action through `include`, at any depth, for two reasons: the include is what adopts the staged files at the Save, so a create form whose pipeline does not reach the step drops every staged file; and `add-stream` runs an action on an unsaved child only when `create` includes it, so without the include the element's requests before the Save are refused with a 400. That include takes `method: "post"`, because a form's GET runs on past the form and the element's own request already draws the zone; `TestIncludes_LoaderActionsRunOnPostOnly` in [../../tests/viewfiles_test.go](../../tests/viewfiles_test.go) checks it for every loader on a parent's add (BUG-344). Not every `html-remote` element is a file field: the hashtag picker is one too, and binds `data.tags` through `path`, which a file field never carries because its step owns the field name. `TestUploadMultiple_WidgetAndStepAgree` in [../../tests/uploadmultiple_test.go](../../tests/uploadmultiple_test.go) is that connection, for every file field in every template here and in each package checked out beside Emissary that holds an `edit-attachments` step, so extend its reach rather than working around it. Earlier on 2026-11-04 this was an `upload-multiple` widget of its own, first with `url` and `staging` URL templates and a `stage-attachment` step on `base-outbox`, then with an `action` option answered by Emissary's `LookupProvider`; all of that is gone.

## A tab holding exactly one textarea takes no `label:`

In a `layout-tabs` form, the tab heading already names a lone field, so a label repeats the same word directly beneath itself and makes the form look like it is missing a field. Keep the `description:` — that explains rather than names. With two or more fields in a tab, every field keeps its label.

## Testing `_hs` behaviors under jsdom: two harness limits look exactly like defects

`on load` never fires, because `_hyperscript.processNode(document.body)` does not produce a load event jsdom-side — no error, no output. And `install <behavior>` never runs the behavior's `init`. **Always run a trivial control first** (a handler that just adds a class) before concluding anything about the code under test. Workaround for the first: swap the trigger for a dispatchable custom event and dispatch it.

## A blind find/replace once renamed a third-party option key

Emissary was called **ghost** before it was called **whisper**. A 2022 rename commit ran a textual `ghost` → `whisper` sweep across the repo and hit SortableJS's own option key alongside the legitimate CSS-class renames, turning `ghostClass:` into `whisperClass:` — an option SortableJS does not have, so the drag placeholder went unstyled and stayed that way for years. When renaming across a tree, exclude vendored code and check that every changed key still belongs to something you own.

## Dark mode is half-built and inert — do not re-flag it

The theme carries partial dark-mode tokens that nothing currently activates. This is a known incomplete feature rather than a bug; leave it alone unless the work is explicitly being picked up.

## Form option templates render against the edited object, not the Builder

A `{{...}}` inside a form element's `options:{...}` (a `validator` URL, an upload's `delete` URL) is compiled by `benpate/form` when the Template loads and rendered by the widget against the object the form edits: the `*model.Stream`, `*model.Group`, `*model.Folder`, and so on. Builder methods such as `.ObjectID` do not exist there, and an ID field such as `.StreamID` is a `primitive.ObjectID` that prints as `ObjectID("...")`. Write `{{.ID}}`, which every model implements as its hex primary key. Step-level `options:[...]` are different: they are still rendered against the Builder. [../../service/template_optionTemplates_test.go](../../service/template_optionTemplates_test.go) renders every form option template in `_embed/templates` against its object (BUG-204); a new model whose Templates use option templates needs a case in its `optionTemplateObject`.

**Only `edit-model-object` and `add-model-object` render against a model object.** An `edit-table` form renders against each row, a Theme form against the Domain, a Widget form against its data map, and a Registration form against the new User's data, so `{{.ID}}` means something different, or nothing, in each. `TestEmbeddedForms_NoOtherOptionTemplates` fails on any option template in those forms; adding one means writing the render test for its data first.

**An option template may reference only IDs and other values that cannot contain a quote.** `benpate/form` builds `install validator(url:'…')` and `show-if` by concatenating the rendered value into a single-quoted hyperscript string. The HTML builder escapes the attribute, but the browser decodes that escaping before hyperscript parses the string, so a `'` in a rendered value ends the string early and injects hyperscript. A hex ID is safe. A field a user can edit, such as a Stream's `Token` or `Name`, is not: pipe it through `urlquery`, which encodes the quote.

## The `if` step's second branch is `else`, and an unknown key is ignored

`model/step/ifCondition.go` reads `condition`, `then`, and `else`. A branch written as `otherwise:` parses without error and is dropped, so a false condition runs nothing and the action answers `200` with an empty body. That is how the `annotation` action showed a blank modal for every annotation whose type was not `ACTOR`. `index` and `notifications` in user-inbox had the same defect until 2026-10-11, so their legacy redirects ran only the `then` branch.

## A journal note is `comment:` on a `save` step, and it names no one

`model/step/save.go` reads `comment` and nothing else, so a note written as `message:` parses and is never stored, and `save-and-publish` and `add-stream` take no note at all. Do not render `{{.Author}}` into one: it is the Stream's `AttributedTo`, not the person saving, and `model.PersonLink` has no `String` method, so the note stored the whole record, email address included. Until 2026-11-07, 34 notes did one or the other (EmissarySocial BUG-345). Write a fixed phrase, such as `comment:"Settings updated"`.

## A click on an icon inside a button has the icon as its target

`event.target` is the innermost element pressed, so a click on an icon-only button lands on the `<i>`, not the `<button>`. `blockselect` used to test only `event.target` for `hx-get`, so a click on the icon of an annotation button inside a newsfeed row also opened the row's `browse` modal, and the two modal requests raced: the Notes view appeared for some posts and not others. Test with `event.target.closest(…)`, limited to elements inside the behavior's own element.

## Never write a remote or user-edited value into a `script` attribute

`html/template` escapes a `script="…"` attribute as plain HTML, not as hyperscript, and the browser decodes that escaping before hyperscript parses it. A URL from a remote document can hold a `'`, so `install refreshPeers(id:'{{$url}}')` let a remote actor's ID end the string and inject hyperscript. A Stream's `Label` and `Summary` are no safer, because any author can type a quote into them. Put the value in a `data-` attribute and have the behavior read it with `@data-…`, as `refreshPeers`, `annotationPeer`, and `Share` in theme-global do. Compare those values in hyperscript rather than building a CSS selector from them, which breaks on `]` and quotes.

## A behavior builds an element from data, never an HTML string

`put` parses a string as HTML, so `select` and `selectGroup` once built each choice with ``put `<option value="${option.value}">${option.label}</option>` ``, and a lookup label could run an `onerror` handler while a value could end its attribute. Both now `make an Option from option.label, option.value` and `add()` it, which sets the label and value as text. Any behavior that writes data into the page must do the same: `make` the element and `set` its properties or `textContent`. `colorpicker._hs` still interpolates its stored value into markup, which is safe only because every colorpicker path today has `Format: "color"`.

## Only four template directories are parsed by any test — add yours to the list

`TestEmbeddedTemplates_HTMLParses` in [service/template_html_parse_test.go](../../service/template_html_parse_test.go) is the only thing that parses `*.html` here with the real funcMap, and it is **scoped to a hand-maintained `dirs` list**, not a full sweep. A directory that is not named there is never parsed by anything, so a syntax error, an undefined funcMap function, or the attribute-context lexer gotcha first appears in front of a visitor.

A full sweep is not possible: some email and layout templates legitimately rely on cross-file variables like `$title` and cannot parse standalone. So every directory that *can* parse standalone has to be added by hand when it is created. `TestEmbeddedTemplates_Validate` does cover every template's `hjson`, which is why a missing directory looks covered — the two tests have very different reach.

## Every `target="_blank"` carries `rel="noopener noreferrer"`, and the Go emitters do too

CodeQL's `js/unsafe-external-link` flags a `_blank` anchor whose `href` is dynamic or off-site and whose `rel` lacks `noopener`, and it runs against `main` on every push. There were 75 of these across 56 template and hjson files, and two more in hjson strings that a truncated listing hid — so check with a tag-level sweep of the whole tree, never a `grep | head`. The attribute goes on **every** `_blank` anchor, not only the ones CodeQL happens to flag, so that a `href="/settings"` that later becomes `href="{{.URL}}"` does not turn into a new alert.

When the anchor already carries a `rel`, **append** — `rel="me"` on the profile links in `user-outbox/sidebar.html` is the IndieWeb identity claim, and replacing it silently breaks rel-me verification for every user. The safe form there is `rel="me noopener noreferrer"`.

The same rule reaches federated content by a different route. `replace.Linkify` and `Content.ApplyLinks` write `<a … target="_blank">` into `content.HTML` **after** `markdown.Sanitize` has run, so they are the last writers of the body every remote reader sees, and the `bluemonday.UGCPolicy` never gets a chance to add a `rel` for them. They emit `rel="noopener noreferrer"` themselves — the convention Mastodon uses on every outbound link — and five test files pin the exact anchor, so a change to either emitter shows up as a test failure rather than a quiet drift.

Vendored `easymde.min.js` still opens three `_blank` links without it; that is upstream's file, and the fix is a version bump, not an edit.

## Every `type: array` in a schema needs a `maxLength`

Rosetta bounds a list's length only through the schema, never in its accessors (decided 2026-10-05; see rosetta's AGENTS.md). An array with no `maxLength` lets one form field such as `feeds.5000000` build a list of 5,000,001 items. Give every array in a template's `schema` or `socialSchema` a `maxLength` sized to its real use.

## Profile settings come from the user's outbox, so every outbox extends `base-outbox`

`/@me/settings/general` renders `settings-general` from the user's own outbox template through the Settings builder's `.OutboxView`, so an outbox without `base-outbox` in its `extends` shows only a "not available" notice where the Profile tabs belong. `service.Template.missingProfileTabs` reports such an outbox on every load, and never halts. List `base-outbox` in every outbox's own `extends`, even one that already gets it through `user-outbox`, so a reader sees it without following the chain.

**Never ship a package that names `base-outbox` before the Emissary release that defines it.** An `extends` that names an unknown template stops the template load and halts first boot.

**Customize Profile through the outbox, never by reusing the ID `user-settings`.** The last definition of an ID wins for every user on the server. `Template.Add` reports a replaced ID, but it still replaces it.

## Go builds the URLs of three email pages, so every outbox gets them from `base-outbox`

Three actions are opened by URLs written in Go, never by a link in a template: the redirect after `/.follower/new` saves a follower (`follow-email-sent`), the link in the confirmation email (`follower-confirm`), and the unsubscribe link and `List-Unsubscribe` header of every email a follower receives (`follower-unsubscribe`). An outbox without them answers each one with 400 "Action is not valid", and nothing reports it. Qwertylicious's outbox defined only its own `follow` until 2026-10-29, so its email signup ended on that 400, and so did both email links. These actions now live in `base-outbox` with the rest of the Follow modal, and `missingProfileTabs` names any outbox that lacks one.

**The Follow modal belongs in `base-outbox`, not `base-intent`.** Stream templates extend `base-intent` too, and these views call Outbox builder methods (`.UserID`, `.DisplayName`, `.IsBridgeBluesky`) that the Stream builder lacks, so every Stream would inherit actions that fail when they render. `follow-email` and `follow-email-sent` were in `base-intent` until 2026-09-17.

## A Profile tab is an htmx swap, never a boosted link

Each tab in `settings-tabs.html` is an `hx-get` to `/@me/settings-<tab>` with `hx-target="#profile-settings"` and `hx-push-url="false"`. Neither the tab nor `#profile-settings` may set `hx-swap`: the swap inherits the body's `innerHTML transition:true`, and a plain `hx-swap="innerHTML"` on either silently turns off the view transition between tabs. A boosted link pushes `/@me/settings-<tab>` into the address bar, and reloading that URL renders a bare fragment inside the outbox's page layout. A form that refreshes after a save must reload only its own card, as `settings-images` does, or the user is sent back to General.

## `settings-general` can draw any Profile tab, so a tab bar cannot trust `.IsAction`

The settings page draws its first tab by running the outbox's `settings-general` with the page's own request, and `base-outbox` makes that action a `switch` on `?tab=`. Under `settings-general`, then, `.IsAction` names the action and not the tab on screen, and a tab bar that highlights by `.IsAction` alone highlights General above the Images form. Nothing errors.

The stock `settings-tabs.html` reads the same parameter, trimmed and case-sensitive like the switch, and accepts only the names its cases list. **The two lists must agree.** An outbox that overrides the action replaces the whole switch, so its cases repeat the stock ones (Qwerty's adds `collections`), and an outbox with a tab bar of its own must read `?tab=` the same way, whether or not it overrides the action. Every tab click is a new request to `/@me/settings-*` with no query, so the parameter only ever affects the first draw.

## `base-outbox` declares `model: Outbox` on purpose

Template validation checks every form path against the template's own schema. Under `model: None` its schema has no User fields, so the Profile forms fail validation. `Outbox` gives it `UserSchema`, and leaving out `templateRole` keeps it out of the profile template picker, which lists by `templateRole: user-outbox`.

## An outbox's schema validates profile saves, and only its `data` strings count

The Outbox builder validates against `model.UserSchemaForTemplate`, which is `UserSchema` plus the String rules the outbox declares under `data`. Every other field keeps `UserSchema`'s rules, even where the outbox redeclares it, and every `data` value stays capped at 4096 characters. A non-String `data` rule is ignored, because `User.Data` holds only strings. An invalid value is dropped, not refused: `form.SetURLValues` logs a field that fails and saves the rest.

## A remote value never goes inside `hx-vals`

`hx-vals` holds JSON inside an HTML attribute. `html/template` escapes a value there for HTML only, so a `"` becomes `&#34;`, the browser decodes it back, and htmx parses JSON in which the value has closed its own string. A remote post whose ID held `","type":"Announce","url":"…` turned a reader's Like into a Boost of a URL the sender chose (found 2026-11-10). Put the value in the link, as `?url={{$url | urlquery}}`, and keep `hx-vals` to constants. `set-response` reads `url` from the link or the body. `TestTemplates_NoFederatedValueInHxVals` checks the folders that draw federated content.

## Bump `?v=` when a global bundle changes

The global stylesheet, JavaScript, and hyperscript bundles are served for an hour (`model.Bundle.GetCacheControl`), and a rolling deploy serves old and new pages side by side. A page that pairs a new bundle with a cached old one breaks without an error: the new JavaScript calls a hyperscript function the old bundle no longer has. So every theme's `includes-head.html` and `includes-foot.html` put one `?v=` number on all three, and a release that changes any file under `theme-global/stylesheet`, `javascript`, or `hyperscript` raises it in the same commit. Themes in other repositories carry their own copy of the number.
