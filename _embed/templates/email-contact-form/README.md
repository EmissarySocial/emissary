# Contact Form Email

Delivers one visitor's contact-form submission to the address its page author configured. Sent from the `submit` action of the `contact-form` Stream template, which reads the visitor's fields with `read-form` and then names this definition in a `send-email` step. Nothing is stored: this message is the only record that the submission happened, which is why a send failure halts the pipeline rather than being logged and skipped.

## Data contract

The email body cannot see the Stream. `StepSendEmail.postTemplateEmail` (build/step_SendEmail.go) renders each of the step's `values:` templates against the Builder and hands `DomainEmail.Send` a flat map, so the templates here see exactly the keys the step supplied plus the four `Domain_*` values that `Send` injects into every email.

| Key | Source | Trust | Used by |
| --- | --- | --- | --- |
| `To` | Stream `data.emailAddress` | page author | `to` |
| `Subject` | Stream `data.emailSubject` | page author | `subject` |
| `ReplyEmail` | visitor | untrusted | `Reply-To`, body |
| `Name` | visitor | untrusted | body |
| `Message` | visitor | untrusted | body |
| `HeaderMessage` | Stream `data.headerMessage` | page author | body |
| `Client_IP` | `Common.ClientIP` | resolved | body |
| `Client_Description` | `sniff`, over `User-Agent` | untrusted | body |
| `Client_Referer` | `Referer` | untrusted | body |
| `Client_UserAgent` | `User-Agent` | untrusted | body |
| `Client_Brands` | `Sec-CH-UA` | untrusted | body |
| `Client_Platform` | `Sec-CH-UA-Platform` | untrusted | body |
| `Client_Mobile` | `Sec-CH-UA-Mobile` | untrusted | body |
| `Client_AcceptLanguage` | `Accept-Language` | untrusted | body |
| `Client_Accept` | `Accept` | untrusted | body |
| `Client_AcceptEncoding` | `Accept-Encoding` | untrusted | body |
| `Client_DoNotTrack` | `DNT` | untrusted | body |
| `Client_PrivacyControl` | `Sec-GPC` | untrusted | body |
| `Domain_Icon`, `Domain_Name` | domain config | trusted | body |

## Sender details

The `Client_*` block is the message's only forensic record. Nothing is stored (D1), so if the recipient wants to know where a message came from — to answer it, to block a source, or to file an abuse report — this footer is the one place that information exists. It describes the **browser**, not the person: an IP address and the headers the client volunteered.

Every one of them is **untrusted**, and in a stronger sense than the visitor's `Name` and `Message`. Those come from a form the visitor filled in; these come from headers the client chose byte by byte, and a script sends whatever it likes. A `User-Agent` claiming to be Safari is a claim, not a fact, and `Client_Description` is `sniff`'s guess *about that claim* — it is there to make the raw string readable, never to be relied on.

`Client_IP` is the one value that is *resolved* rather than merely read: `Common.ClientIP` asks the server's configured `clientIpStrategy` (`REMOTE-ADDR`, `RIGHTMOST-TRUSTED-COUNT`, or `SINGLE-IP-HEADER`), so it is correct behind a reverse proxy where `RemoteAddr` would be the proxy — identical for every visitor on the site. It is also the only value that appears in an `href`, linking to `ipinfo.io` so the recipient can look it up themselves. That is safe because every `realclientip` strategy returns `""` for anything it cannot parse as an address, so the value is either empty or an IP. Following the link is the recipient's choice and sends nothing from the server; a deployment that prefers a different lookup (`abuseipdb.com/check/`, say) changes the one `href` by overriding this folder.

`Client_Referer` is deliberately rendered as **text, not a link**. Its value is whatever the client sent, so `javascript:…` is a legal thing for it to contain. `html/template` would neutralize that in a URL context — by rewriting the value to `#ZgotmplZ`, which hides from the reader the exact thing the row exists to show them.

Length is bounded in Go at 256 runes per value, marked with an ellipsis when cut. Go's HTTP server caps the total header block but no individual header, so an unbounded accessor would let a hostile client dictate the size of a message the owner has to open. These are truncated rather than rejected — the opposite of D10, which governs the visitor's *message*, where silent shortening is unrecoverable. Discarding a submission because a browser sent a long header is the false positive FORM-SPAM-PREVENTION D3 forbids.

Each row is wrapped in `{{with}}`, so a value the client did not send contributes no row at all — a Firefox or Safari visitor sends no `Sec-CH-UA` family, and that absence is itself a signal. The `IP Address` row uses `{{if}}` and always renders, so the block can never be a heading with nothing under it.

**Do not guard this block with `or`.** Emissary's funcMap inherits `rosetta/funcmap.All()`, which shadows the `and`/`or` builtins with `func(...bool) bool`; `{{if or .Client_IP .Client_UserAgent}}` fails at render time with `expected bool; got string`, and on an absent key with `invalid value; expected bool`. `if` and `with` are template keywords rather than functions, so they still accept any type and still tolerate a missing key — which the body depends on, since only `to` and `headers` carry `missingkey=error` (D15).

`To`, `Subject`, and `ReplyEmail` are checked when Templates load: `ServerEmail.RequiredKeys` walks the `to` and `headers` parse trees, and `service.Template.validateTemplates` rejects any `send-email` step that does not supply every key it finds. The body and subject keys are **not** covered by that check, because `subject` and `body` are deliberately lenient about missing keys (model.NewEmail applies `missingkey=error` only to `to` and `headers`). `TestContactFormEmail_KeysAreInTheContract` in service/serverEmail_test.go closes that gap for this email by walking both parse trees and asserting every key they reference appears above.

## Who controls what

`From` is never in the table, and never can be. `ServerEmail.Send` builds it from the domain owner's configured address, and `From`, `Sender`, and `Return-Path` are on the `reservedHeaderNames` denylist, so a `headers:` block naming one is rejected at load. This matters beyond spoofing: the sending provider requires the domain's own address, so a drifting `From` means non-delivery.

The visitor influences exactly one header, `Reply-To`, which is the reason the `headers:` block exists at all. There is no subject field on the form — the subject comes from the page, so a visitor cannot title a message "Re: your overdue invoice" in the owner's inbox. `Reply-To` carries an address nobody has verified; hitting reply mails whoever the visitor typed.

An empty `ReplyEmail` is a present-but-empty key rather than a missing one, so `applyHeaders` omits the header instead of emitting a malformed one. In practice the `read-form` schema requires the field and validates it as an email address, so an empty value never reaches here.

## Trust boundary in body.html

Two trust levels share one document, and the distinction is load-bearing rather than stylistic.

The page author's `HeaderMessage` is Markdown, rendered through the `markdown` helper — the one helper in the funcMap that sanitizes its own output before returning `template.HTML` (tools/templates/functions.go).

The visitor's `Name`, `ReplyEmail`, and `Message` are interpolated plainly so that `html/template` escapes them. They must never be piped through `markdown`, `htmlMinimal`, `highlight`, or any other helper with an `HTML`, `CSS`, or `HTMLAttr` return type — every one of those tells the template engine the value is already safe. `Message` renders inside a `white-space:pre-wrap` block so the visitor's line breaks survive without any markup being introduced to carry them. `TestContactFormEmail_EscapesVisitorInput` pins this.
