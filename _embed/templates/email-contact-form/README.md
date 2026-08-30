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
| `Domain_Icon`, `Domain_Name` | domain config | trusted | body |

`To`, `Subject`, and `ReplyEmail` are checked when Templates load: `ServerEmail.RequiredKeys` walks the `to` and `headers` parse trees, and `service.Template.validateTemplates` rejects any `send-email` step that does not supply every key it finds. The body and subject keys are **not** covered by that check, because `subject` and `body` are deliberately lenient about missing keys (model.NewEmail applies `missingkey=error` only to `to` and `headers`). `TestContactFormEmail_KeysAreInTheContract` in service/serverEmail_test.go closes that gap for this email by walking both parse trees and asserting every key they reference appears above.

## Who controls what

`From` is never in the table, and never can be. `ServerEmail.Send` builds it from the domain owner's configured address, and `From`, `Sender`, and `Return-Path` are on the `reservedHeaderNames` denylist, so a `headers:` block naming one is rejected at load. This matters beyond spoofing: the sending provider requires the domain's own address, so a drifting `From` means non-delivery.

The visitor influences exactly one header, `Reply-To`, which is the reason the `headers:` block exists at all. There is no subject field on the form — the subject comes from the page, so a visitor cannot title a message "Re: your overdue invoice" in the owner's inbox. `Reply-To` carries an address nobody has verified; hitting reply mails whoever the visitor typed.

An empty `ReplyEmail` is a present-but-empty key rather than a missing one, so `applyHeaders` omits the header instead of emitting a malformed one. In practice the `read-form` schema requires the field and validates it as an email address, so an empty value never reaches here.

## Trust boundary in body.html

Two trust levels share one document, and the distinction is load-bearing rather than stylistic.

The page author's `HeaderMessage` is Markdown, rendered through the `markdown` helper — the one helper in the funcMap that sanitizes its own output before returning `template.HTML` (tools/templates/functions.go).

The visitor's `Name`, `ReplyEmail`, and `Message` are interpolated plainly so that `html/template` escapes them. They must never be piped through `markdown`, `htmlMinimal`, `highlight`, or any other helper with an `HTML`, `CSS`, or `HTMLAttr` return type — every one of those tells the template engine the value is already safe. `Message` renders inside a `white-space:pre-wrap` block so the visitor's line breaks survive without any markup being introduced to carry them. `TestContactFormEmail_EscapesVisitorInput` pins this.
