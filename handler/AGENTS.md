# handler — Notes for AI Agents

HTTP handlers for the web UI, JSON API, attachments, OAuth, sign-in, SSE, checkout, and the setup console; see [README.md](README.md) for the layout. The Mastodon-compatible API has its own notes in [mastodon/AGENTS.md](mastodon/AGENTS.md) — this file does not repeat them. The shared ActivityPub receive/collection machinery is documented in [activitypub/doc.go](activitypub/doc.go); the four `activitypub_*` packages build on it.

## Authentication happens per-wrapper, not in global middleware

The global `steranko.Middleware` in [../server.go](../server.go) is commented out on purpose. Every route gets its identity through a `WithXXX` wrapper in [wrappers.go](wrappers.go), which builds a `steranko.Context` from the request. Know which family a route uses before touching it:

- **Steranko session (cookie or Bearer JWT → `model.Authorization`)**: `WithAuthenticatedUser`, `WithAuthenticatedAPI`, `WithOwner`, `WithIdentity`, `WithPrivilege`. `getAuthorization` (in [utilities.go](utilities.go)) returns an ANONYMOUS authorization on any parse failure — it never errors — so the refusal must come from the wrapper's own `IsAuthenticated`/`DomainOwner` check, and any new handler that reads `getAuthorization` directly must gate explicitly.
- **OAuth grant on top of the session**: `WithOAuthUser` additionally requires the URL's `:userId` to match the token's user, and loads the live grant by the access token's grant-ID claim — a missing grant record means the (still cryptographically valid) token was revoked, and the request gets a 401.
- **ActivityPub HTTP signatures**: `WithActor` and friends, via `resolveSignedActor`. Three outcomes, kept strictly apart: no signature = anonymous, valid signature = that Actor, INVALID signature = 401 for the whole request. Never collapse the third case into the first — a silently-anonymous response sends the peer's operator hunting a permissions bug that does not exist.
- **JWT carried in the URL/query**: `WithMerchantAccountJWT` (checkout response) and `GetIdentitySigninWithJWT` (guest OTP link) parse a signed JWT from the request itself; these are the only paths where a link click authenticates.

## WithAuthorizedActorAndUser is wired to no route

It is the authorized-fetch ("secure mode") gate, kept for a future Domain-level setting. Its unit test drives the gate directly, so green CI says nothing about reachability — check [../server.go](../server.go) for a route's actual wrapper before assuming it runs. Inside it, a blocked actor gets a 404 identical to a missing user (probing must not tell them apart), and MUTE or LABEL matches never gate (a muted actor must not be able to detect the mute).

## GET gets a read-only session; everything else gets a transaction

`WithFactory` opens a plain read-only `data.Session` for GET, and wraps every other method in `factory.WithTransaction` — NOT `factory.Server().WithTransaction` — because the factory variant attaches the post-commit task spool, so queue tasks publish only after the commit. Any handler-level write that bypasses this loses that guarantee.

## Theme templates rendered by a handler take a `build.Theme`, never a map

Sign-in, sign-out, guest sign-in, password reset, checkout claim, and the OAuth consent page bypass the action pipeline, but they render the same Theme partials every other page does. Those partials call accessors on their dot, and `html/template` resolves them at RENDER time — so a dot that is missing one does not fail to build, it 500s the page in production. The two call shapes fail differently, which is what hides the problem: `{{.IsIndexable}}` against a map silently yields nil, while `{{.ThemeData "token"}}` — the same lookup with an argument — is a hard error, because arguments are accepted only for a real method. Adding a single argument to a partial's accessor breaks every non-conforming dot at once.

Route these through `executeThemeTemplate` ([template.go](template.go)). Its `page` parameter is `any` on purpose — a typed signature could only prove that two methods exist, not that the dot is the right one (`build.Common` satisfies any such interface and would quietly make an auth page indexable), and it would be the only typed seam in a template layer that types nothing else. `build.Theme` is the default dot and owns the `noindex` rule; `build.PasswordReset` embeds it and adds the reset-code User; `build.OAuthAuthorization` embeds it and adds the OAuth client. The old map-based dots were `noindex` only by accident — a map miss yields nil, and `not nil` is true. [themeTemplate_test.go](themeTemplate_test.go) is what enforces this instead: it maps each template to its dot and walks the parse tree (partials included) to check both name and arity, and a second test fails if any template rendering `includes-head` is missing from that map. Add a row whenever a handler renders a Theme template — the test will tell you by name if you forget.

## Every caller-supplied or remote URL is guarded before it reaches an href or redirect

- `uri.IsSafeRedirectURL` is the gate for link targets: `getIntent_Continue` ([intent_continue.go](intent_continue.go)) replaces an unsafe URL with `/@me` BEFORE both the displayed text and the href, so the page always shows the URL it links to; `safeIntentURL` ([intent_header.go](intent_header.go)) returns `""` for unsafe values so the html builder omits the attribute entirely. Remote actor profile/icon URLs count as untrusted — a fetched actor can serve `javascript:` URLs.
- `calcNextURL` ([signin.go](signin.go)) reduces the `next` parameter to a same-origin path via `uri.PathAndQuery` and refuses `/signin`/`/signout` loops.
- Remote object summaries/content on intent pages pass through `convert.SanitizeHTML` before `InnerHTML` — sanitize, don't escape, so legitimate post markup survives.

## Attachments serve inline only when FFmpeg re-generated the bytes

`setAttachmentHeaders` ([attachment.go](attachment.go)) is the whole defense against stored XSS via uploads: a file serves inline with a real Content-Type only when `attachment.CanServeInline()` says MediaServer re-encodes it through FFmpeg (FFmpeg cannot emit script); everything else is `application/octet-stream` plus a forced-download `Content-Disposition`, with `nosniff` and `default-src 'none'; sandbox` on all four attachment handlers. Never let `http.ServeContent` pick the type — it would derive it from the REQUEST URL's extension, letting the visitor choose `text/html`. The `filespec` passed to `setAttachmentHeaders` must be the same one handed to `MediaServer.Serve`, or the headers describe a different file. A `mediaserver.Serve` failure is deliberately downgraded to 404 so one broken attachment does not read as a server fault.

## Checkout never authenticates a buyer from a Stripe email

Stripe does not verify customer emails, so `GetCheckoutResponse` ([checkout.go](checkout.go)) must never reach `Steranko.SetCookie` when its only identity evidence is the email Stripe reported — that would be an account takeover of any pre-existing Identity on that address. A buyer already signed in as the Privilege's Identity is redirected to `/@guest`; anyone else gets a guest-code (OTP) email and the `checkout-claim` page. The ONLY handler that sets a guest cookie is `GetIdentitySigninWithJWT` ([identity.go](identity.go)), which requires the signed OTP link. `GetCheckout` locks `customerEmail` to a signed-in guest's verified address so they cannot purchase under a different email.

## The /@:userId/sse route is a deprecated alias — leave it unguarded

`ServerSentEvent_Me` ([sse.go](sse.go)) never reads the `:userId` param; it streams the signed-in User's events from the auth cookie, so the alias cannot leak another user's stream. Do NOT add an ownership check that 404s on mismatch: the 404 is itself a logged error, and the alias exists precisely to stop the log flood from stale pages that reconnect forever (an old tab after switching accounts triggers it constantly). Remove the alias route in [../server.go](../server.go) only once pre-move pages and third-party skins have aged out.

## The setup console's security model is two middlewares and nothing else

Setup routes (`setup_*.go`) are registered only in setup mode, behind `mw.Localhost()` (exact-hostname match) and `mw.CrossOriginProtection()` — there is no authentication. The fragment endpoints (`SetupDomainGet`, `SetupServerGet`, `SetupDomainUsersGet`) redirect non-htmx requests to their parent page BEFORE any factory access; keep that guard first — [setup_fragments_test.go](setup_fragments_test.go) passes nil factories and fails if it ever moves below a factory dereference. The first-boot owner account gets a convenience password only when the domain `IsLocalhost()` (see `createOwner` in [../service/domain.go](../service/domain.go)); a known default credential is never shipped on a public host, and that gate is intentional, not a bug.

## Outbound fetches go through clients that carry the SSRF policy

`factory.Camper()` and `factory.ActivityStream()` thread the domain's `AllowPrivateIPs` setting into the remote transport, whose resolved-IP dial guard is the authoritative SSRF defense. Never call `digit.Lookup` (or build a bare remote client) directly from a handler — a bare lookup silently loses both the private-IP policy and the HTTP cache, which breaks every local-dev federation flow and weakens production. `PostProxyURL`'s `uri.IsLocalURL` check ([proxy.go](proxy.go)) is only a fast, friendly reject; it does no DNS resolution and must not be treated as the guard.

## HEAD and GET derive their headers from the same place

Content negotiation (HTML page vs. ActivityStreams document from one URL) and the headers it implies live in [../tools/headers](../tools/headers/headers.go); hannibal owns the ActivityPub reading of the Accept header. [headers_parity_test.go](headers_parity_test.go) pins RFC 9110 §9.3.2 parity across an Accept matrix — new dual-representation handlers must route through `headers.SetAll`/`headers.SetVariant`, not hand-set Content-Type/Vary.

## OAuth server rules worth knowing before "simplifying"

- A public client (empty `client_secret`) must have bound its authorization code to PKCE; a confidential client authenticates with a constant-time secret check (`oauthClient.ValidateSecret`). Both live in `postOAuthToken_authorizationCode` ([oauth-server.go](oauth-server.go)).
- `PostOAuthRevoke` deliberately lets a public client revoke with no secret: possessing the token string is itself the proof, and RFC 7009 treats an unknown token as a successful revocation.
- `model.OAuthUserTokenRequest.Validate` is never called from these handlers; don't assume it gates anything.

## Assorted facts that look wrong but aren't

- The Mastodon API routes are not registered: `toot.Register` in [../server.go](../server.go) is commented out. [mastodon.go](mastodon.go) is the glue that builds the `toot.API` when it is re-enabled.
- Discovery/intent failure pages (`PostOStatusDiscover`, `outboundIntentError`) return HTTP 200 with a human-readable message — they are htmx/user-facing response targets, not API errors.
- `GetSignOut` renders a confirmation page only; the actual sign-out happens in `PostSignOut`, so state never changes on GET. `PostSignOut` may pop a stacked admin "backup" session (masquerade, see [signin_masquerade.go](signin_masquerade.go)) instead of fully signing out.
- `PostResetPassword` returns a uniform success page for "email sent" and "user not found", but an honest error page when the account exists and SMTP fails — a mild enumeration signal accepted on purpose so locked-out members aren't stranded by a broken mail server.
- SSE connections cap at 30 days by design — a reclamation backstop, not a timeout to shorten.
