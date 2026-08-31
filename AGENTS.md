# Emissary — Notes for AI Agents

See [README.md](README.md) for what Emissary is and [build/README.md](build/README.md) for how templates and action pipelines fit together. These are the repo-wide rules that are not visible in the code.

Package-specific notes live in the nearest `AGENTS.md` — currently [service](service/AGENTS.md) and [handler/mastodon](handler/mastodon/AGENTS.md). Put a lesson in the most specific file that covers it; this one is only for rules that span packages.

## Never re-purpose an upgrade slot number

[queries/upgrade.go](queries/upgrade.go) holds an ordered slice of `upgrades.VersionN` functions, and **the slice index is the `databaseVersion` written to each Domain record**. A deployed server that already recorded version N will never re-run slot N, so changing what slot N does silently skips the migration on every existing install while running it on new ones. Only ever append a new slot; to fix a bad migration, add the correction as the next slot.

## Enqueue background work through `postcommit.Publish`, never `queue.Publish`

Inside a transaction, a task published directly to the queue can be consumed before — or instead of — the commit that makes its data visible, so the worker reads a record that does not exist yet. [tools/postcommit](tools/postcommit/postcommit.go) spools tasks onto the transaction's context and publishes them FIFO only after a successful commit; a rollback drops them. Non-transactional callers (GETs, schedulers, startup) fall through to an immediate publish, so `postcommit.Publish` is always the correct call. Pair it with `postcommit.WithTransaction` rather than `server.WithTransaction` wherever a transaction may enqueue anything.

## `Response.SetResponse` is the only writer of Response records

[service/response_.go](service/response_.go) centralizes the create/update/delete decision for likes, dislikes, and other responses, plus the target resolution and counter bookkeeping that go with it. There are two entry paths into it (a remote activity and a self-loopback), but still one writer. Writing a `model.Response` from anywhere else desynchronizes the counters and defeats the unique index that keeps duplicates out.

## Local MongoDB requires `?directConnection=true`

A Go client connecting to a single-node replica set from the host will otherwise try to reach the node by its advertised replica-set name and hang until timeout, with no useful error. Every local connect string — config, tests, `mongosh` one-liners — needs the flag.

## Never run `go mod tidy` while a local `replace` is in `go.mod`

Emissary regularly consumes `benpate/*` and `EmissarySocial/*` libraries from local working copies while a fix waits for a tag. `go mod tidy` rewrites `go.sum` and the require block against those local trees, which produces a `go.mod` that cannot build for anyone else and is easy to commit by accident. If tidy is genuinely needed, drop the replaces first — and never keep its rewrite silently.

## An email recipient never comes from the request

A `send-email` step reaches the outside world on behalf of a visitor who may be anonymous, which makes the `To:` value the line between a contact form and an open relay. It must resolve from the Stream — `{{.Data \`emailAddress\`}}`, set only through an author-gated settings form — and never from anything the sender controls. The builder that renders those step arguments also exposes `.QueryParam` and the posted form, so writing `To: "{{.QueryParam \`email\`}}"` compiles, loads, validates, and ships a relay. Nothing in the code stops it; this rule is the whole enforcement.

The same reasoning covers `ReplyEmail`, which *does* carry visitor input: it is a reply-to hint on a message going to a fixed recipient, not a destination. Anything that selects a destination stays on the Stream.

[StepSendEmail](build/step_SendEmail.go) also halts the pipeline when a send fails, rather than reporting and continuing. A web-form message exists only in flight — nothing is written and nothing is queued — so a swallowed error returns a success page to a visitor whose message reached nobody. `DomainEmail.Send` treats an unconfigured SMTP connection the same way, for the same reason.

## The template funcmap has helpers that emit unescaped HTML

`markdown`, `highlight`, `icon`, and their siblings in [tools/templates/functions.go](tools/templates/functions.go) return `template.HTML`, which tells `html/template` the value is already safe. `markdown` earns that by sanitizing; `highlight` does **not** — it returns its input verbatim. Any new helper with an `HTML`/`CSS`/`HTMLAttr` return type is a trust boundary, so sanitize inside the helper and check every call site before pointing one at federated or user-supplied content.

## An off-site hop needs `forward-to`, or a `redirect-to` that knows it is off-site

Sending a visitor to another URL has two mechanisms and they are not interchangeable. An HTTP redirect is followed by whatever transport made the request: a browser navigates the whole document, but htmx's XHR follows the redirect *inside* the request and swaps the result in as a fragment — which CORS makes impossible across origins, so the click silently does nothing. The `Hx-Redirect` header is executed by htmx itself and always navigates the document, but it is inert for a plain `<a href>`, which lands on a blank 200. Neither failure raises an error anywhere.

Navigation links routinely carry **both** attributes (`<a href="/x" hx-get="/x">`), so both paths must work. [build/navigation.go](build/navigation.go) owns that decision for every step — `redirect-to` means "the content lives at another URL" and `forward-to` means "the visitor goes somewhere else", and each falls back to the other's mechanism where its own cannot work. A Template must never branch on `.IsPartialRequest` to work around this; if a case is not handled, fix the helper.

## Templates are data, not code — a stale copy will not announce itself

Templates in [_embed/templates](_embed/templates/) are embedded at build time, but a server can also load template folders from Git or disk. Those copies are cached, so an edit to a template's actions, states, or roles may need a restart before it takes effect, and a stale external copy silently keeps serving the old pipeline. When a template change appears to do nothing, confirm which copy is actually being served before debugging the Go code.

## `.card` carries `container-type`, so it collapses inside a shrink-to-fit box

`.card` in [theme-global/stylesheet/03-widgets-card.css](_embed/templates/theme-global/stylesheet/03-widgets-card.css) sets `container-type: inline-size` so card contents can use the design system's `@container` queries. That also applies inline-size containment, which sizes the box **as if it had no contents** — its children stop contributing to its intrinsic width.

Put a `.card` inside anything that sizes shrink-to-fit (an absolutely positioned box with only `right`/`bottom` set, a float, an inline-block, a grid/flex item sized to content) and the parent has nothing to size around: the card collapses to its own padding. Nothing errors — a block card renders as a narrow vertical ribbon, and a flex card is worse, because its row does not wrap and simply runs off the edge of the screen where it cannot be seen or reached.

**`width: max-content` does not fix it.** Containment zeroes the very intrinsic sizes that `max-content` resolves against, so the card stays collapsed — the circularity is what container queries have to forbid, not an oversight. The fix is `container-type: normal` on that specific card, which is safe whenever nothing inside it queries its own size — see `.floating-menu` in [theme-minimal/stylesheet/01-layout.css](_embed/templates/theme-minimal/stylesheet/01-layout.css). Do not remove `container-type` from `.card` itself; the `@container` queries throughout `05-*.css` depend on it.

Styling reached through an element selector has the mirror-image problem: a theme that styles its nav items as `nav a { … }` silently loses all of it the moment those links move outside `<nav>`. Check where a rule's scope actually starts before relocating markup — `.nav-item` looks like the hook for this and is not; no stylesheet selects it.
