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

`theme-global/resources/` carries both the current versioned folders (`hyperscript-0.9.93/`, `htmx-1.9.12/`, `sortable-1.15.7/`) and unversioned or older legacy ones. A theme referencing a legacy path gets the old engine, which cannot parse the shared behaviors bundle — bare `focus` is 0.9.93+ — so the whole bundle dies with "Expected 'end' but found 'focus'" and every behavior (Modal, tabs, menu, multiselect) stops installing. The legacy folders are kept deliberately, because **database-stored themes cannot be scanned** and a missing folder 404s into `Sortable is not defined`. Check which folder a theme actually references before debugging a dead behavior.

## Third-party bundles are not self-contained just because they are vendored

EasyMDE 2.21.0 fetches FontAwesome from a CDN on construction unless `autoDownloadFontAwesome: false` is set. Assume any vendored editor or widget issues its own runtime requests, and check for them rather than assuming the bundle is the whole dependency.

## `turboclick` on a CONTAINER is intentional

`theme-global/javascript/turboclick.js` fires a synthetic `click()` on mousedown against **the element actually pressed**, not the nearest `.turboclick` ancestor, so nested links, buttons, `hx-get`, and hyperscript `on click` all fire as they would naturally. `class="turboclick"` on a container (modal header, sort-form rows) is therefore safe — do not "fix" it by moving the class onto children. Two guards must stay: the `.folder-handle` bail, without which pressing a SortableJS drag handle navigates instead of dragging (it was dropped once in a cleanup commit while the doc comment still promised it), and the left-button-only, no-modifier test that preserves open-in-new-tab. If a click does nothing inside a turboclick region, the synthetic-click targeting is not the likely cause; look at the htmx swap rule above first.

## A struct dot reaching `includes-head` must implement `IsIndexable()`

`includes-head` evaluates `.IsIndexable`. `html/template` treats a missing field on a **struct** dot as a hard execution error (HTTP 500), while a missing key on a **map** dot silently evaluates nil. Any new lightweight builder that is a struct, does not embed `build.Common`, and renders through `includes-head` must implement `IsIndexable() bool`. `build.OAuthAuthorization` 500'd every `GET /oauth/authorize` this way, the second builder to hit it after `model.Domain`.

## A Template's create seed and its schema default are linked by nothing but a test

`data.width` is `["FULL","LARGE","MEDIUM","SMALL"]` defaulting to `FULL`. Adding `FULL` above `LARGE` silently redefined `LARGE` from 100% to 75%, so four article templates that seeded `LARGE` on create started producing 75%-wide articles. The seed and the default live in different files and nothing connects them — `TestLayoutControls_CreateSeedsMatchWidthDefault` is that connection, so extend the test's reach rather than working around it.

## A tab holding exactly one textarea takes no `label:`

In a `layout-tabs` form, the tab heading already names a lone field, so a label repeats the same word directly beneath itself and makes the form look like it is missing a field. Keep the `description:` — that explains rather than names. With two or more fields in a tab, every field keeps its label.

## Testing `_hs` behaviors under jsdom: two harness limits look exactly like defects

`on load` never fires, because `_hyperscript.processNode(document.body)` does not produce a load event jsdom-side — no error, no output. And `install <behavior>` never runs the behavior's `init`. **Always run a trivial control first** (a handler that just adds a class) before concluding anything about the code under test. Workaround for the first: swap the trigger for a dispatchable custom event and dispatch it.

## A blind find/replace once renamed a third-party option key

Emissary was called **ghost** before it was called **whisper**. A 2022 rename commit ran a textual `ghost` → `whisper` sweep across the repo and hit SortableJS's own option key alongside the legitimate CSS-class renames, turning `ghostClass:` into `whisperClass:` — an option SortableJS does not have, so the drag placeholder went unstyled and stayed that way for years. When renaming across a tree, exclude vendored code and check that every changed key still belongs to something you own.

## Dark mode is half-built and inert — do not re-flag it

The theme carries partial dark-mode tokens that nothing currently activates. This is a known incomplete feature rather than a bug; leave it alone unless the work is explicitly being picked up.
