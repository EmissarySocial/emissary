# tools — Notes for AI Agents

The packages under `tools/` are small, self-contained leaf libraries — see [README.md](README.md) for the collection overview and [../AGENTS.md](../AGENTS.md) for repo-wide rules. This one file covers all of the subpackages; do not add AGENTS.md files inside individual subpackages.

## Cross-cutting rules

- **Dependencies point outward only.** `service`, `build`, `handler`, and `domain` import tools packages; nothing here may import them back. A tools package may use other tools packages, `benpate/*` libraries, and (sparingly) `model` — only [convert](convert/) and [dataset](dataset/) do today. A helper that needs a service, Factory, or database session belongs in `service/`, not here.
- **A `.gox` file is deliberately parked code**, kept out of the build by its extension ([ascache/consumer.gox](ascache/consumer.gox)). Don't rename it to `.go`, delete it as junk, or try to fix its syntax.
- **Some packages are intentionally unwired.** `asstrict`, `ascontextmaker`, and `jsontemplate` currently have no importers anywhere in the repo. Don't assume they run, and don't wire them in (or delete them) as a drive-by.

## The as* client stack

- **The as* packages are `streams.Client` middleware, assembled in exactly one place**: `newClient` in [../service/activityStream.go](../service/activityStream.go). Outermost to innermost: ashash → asrules → ascache → ascacherules → asnormalizer → assanitizer → tagspub → bridgyfed → webfinger → tombstone → activitypub. The order is load-bearing: asrules sits above ascache so per-viewer verdicts are never written to the shared cache, ascacherules sits directly above ascache so the Cache-Control it rewrites is what the cache stores by, and assanitizer/asnormalizer sit below the cache so the cache at rest only ever holds sanitized, normalized documents.
- **Any wrapper whose `Load` re-loads a different URL must spread the options: `innerClient.Load(otherURL, options...)`.** Writing `options` without `...` compiles fine but passes the slice as a single `any`, silently dropping options like `ascache.WithWriteOnly()` — the observed failure mode is HTTP-signature verification against a stale cached public key. When touching any wrapper, grep for `\.Load([^)]*options)` (missing spread); the fragment-resolving path in [ashash](ashash/client.go) is the hottest spot because key lookups load fragmented URLs.
- **ascache forced reloads carry a default one-minute cooldown.** `WithWriteOnly` bypasses the cached copy, and a remote peer can provoke such a reload at a URL of its choosing, so `NewLoadConfig` applies `defaultMinAge` unless the caller passes `WithMinAge` explicitly. Opt out with `WithMinAge(0)` only when an uncapped refresh is genuinely required.
- **The actor cache ceiling in ascacherules is a security bound, not a performance knob.** An Actor document carries the public key that authenticates everything the Actor sends, so its cache window is also the window in which a revoked key keeps being accepted. See `actorMaxAge` in [ascacherules/client.go](ascacherules/client.go) before "tuning" it.
- **assanitizer is the reserved-namespace trust boundary, and it is also called outside the stack.** POSTed activities never traverse the client stack, so the inbox funnel calls `assanitizer.Strip` directly ([../handler/activitypub/receiveRequest.go](../handler/activitypub/receiveRequest.go), [../handler/activitypub_user/outbox_.go](../handler/activitypub_user/outbox_.go)) and `StripKeys` for bto/bcc ([../service/inbox.go](../service/inbox.go)). `Strip` matches key prefixes (reserved namespaces like `emissary:`); `StripKeys` matches exact names — using prefix matching on a bare name like `bto` would also claim any future property that merely starts with those letters. Both mutate containers in place; DeepCopy first if anything upstream still reads the original.

## postcommit

- The repo-wide rule (always `postcommit.Publish`, paired with `postcommit.WithTransaction`) is in [../AGENTS.md](../AGENTS.md). Local details: `Publish` is fire-and-forget — errors are reported, never returned; a nil session is reported as a defect and falls through to an immediate publish; a nil queue is a silent no-op for test harnesses. The `spool.Reset()` at the top of the `WithTransaction` callback looks redundant but is not: the mongo driver may run the callback multiple times on transient errors, and only the winning attempt's tasks may survive to publish.

## markdown

- **This is the single Markdown converter and the single sanitization policy for markdown-derived HTML** — Stream content, user summaries, widget saveSteps, and the `markdown` template function all route through it; see [markdown/README.md](markdown/README.md). Never call goldmark directly (`WithUnsafe` is only safe because `Sanitize` always follows) and never build a private bluemonday policy for served content.
- **The `markdown` entry in [templates/functions.go](templates/functions.go) deliberately overrides rosetta's helper** with the sanitizing one above. Removing that override reintroduces rosetta's unsanitized converter into every HTML template. The repo-root notes cover the wider funcmap trust boundary (helpers returning `template.HTML`).
- **[convert/sanitize.go](convert/sanitize.go) is not a rival converter.** `SanitizeHTML`/`SanitizeText` sanitize inbound RSS/microformat content with a deliberately plainer policy (no iframes). Don't merge the two policies without deciding which trust level you are changing.

## Template renderers for non-HTML output

- **templatemap executes `text/template` with zero escaping.** It exists for designer-authored templates stored in Template definitions (`model.Template.SearchOptions`); never point it at untrusted input or JSON output.
- **jsontemplate is the sanctioned way to template JSON** (renders through `html/template` inside a discarded `<script>` wrapper so string values are contextually escaped, then unmarshals — leniently via hjson by default, strictly with `WithStrictMode`). It currently has no callers, but if you need to render JSON from a template, use it rather than raw `text/template`, which lets substituted values alter the JSON structure.

## emojikey

- **This package displays EmojiKeys; it never computes them.** The MLS client is the only party that computes fingerprints (SHA-256 of the KeyPackage's leaf signature public key), and the server stores the resulting `summary` verbatim; `Parse` only annotates it for display. Server-side computation was removed on purpose — reintroducing it would drift from the client/Bonfire interop contract, and verification must always recompute client-side from KeyPackage content, never trust the served summary. The canonical emoji table lives in the client (`conversations-mls` `emojikeys.ts`); [emojikey/emojis.go](emojikey/emojis.go) is a display-only mirror. See [emojikey/README.md](emojikey/README.md).

## camper

- **Camper makes outbound WebFinger/NodeInfo fetches, so it is SSRF surface.** Private-IP lookups are refused by default; the factory threads `WithAllowPrivateIPs` and the caching `WithRoundTripper` middleware at construction ([../service/factory.go](../service/factory.go)). Any new outbound call inside camper must ride `camper.options` so those policies apply. The Option API is `WithRemoteOption` / `WithRoundTripper` / `WithAllowPrivateIPs`; the `WithClient` example in [camper/README.md](camper/README.md) predates it.

## stripeapi

- **stripeapi calls the Stripe REST API through `benpate/remote` and imports `stripe-go` only for its response structs.** Keep new endpoint wrappers on that pattern so they inherit remote's transport policies instead of stripe-go's own client. `ConnectedAccount("")` is a deliberate no-op, letting direct-key and connected-account calls share one code path.

## honeypot

- **`Validate` reads the request body through `re.ReadRequestBody`, which puts back a fresh reader.** That restore is load-bearing: form binding after `Validate` still sees the body. Note that despite the README's phrasing, `Validate` only rejects populated honeypot fields; it does not check that required fields are present.
