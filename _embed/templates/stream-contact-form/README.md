# Contact Form

A page that lets visitors send the site owner a message by email. The visitor fills in their name, email address, and a message; Emissary emails it to the address the page author configured and shows a confirmation. **Nothing is stored** — the email is the only record that the submission happened, which is why a failed send halts the pipeline and surfaces an error rather than being logged and skipped.

## What the page author configures

Everything except the visitor's three fields: the recipient address, the subject line, the prompt shown above the form, a header message prepended to every email, and the confirmation shown after sending. All of it is per-page, so one site can run a contact page, a support page, and a booking page with different wording and different recipients.

The settings form is the `edit-form` action, rendered inline by `edit.html` rather than in a modal — the same shape `stream-redirect` uses, down to the Save button living in the menu bar while the form hides its own. Saving runs `save-and-publish`, so there is no separate Publish step and no Publish button; the only way a form stays unpublished is to never save it.

Every one of those settings is `required` in the schema. A contact form missing any of them is broken in a way the visitor discovers rather than the author: no recipient means the send fails outright, and an empty prompt or confirmation means a blank page. `Stream.Save` does not validate the template schema, so the requirement is enforced where the author actually types — the settings form — while `create` seeds everything it sensibly can and leaves only the recipient blank. The Delete link is hidden while `.IsNew`, since there is nothing yet to delete.

The settings form is grouped into three tabs by who reads each value: **Page Content** is what the visitor sees before submitting, **Email Content** is what the recipient receives, and **Confirmation** is what the visitor sees afterward.

The prompt, header message, and confirmation accept Markdown, rendered through the `markdown` helper that sanitizes its own output. The subject line does not: it is a plain string that reaches the email's `subject` template unchanged.

## What the visitor cannot influence

Only one header, `Reply-To`, which carries the address they typed so the recipient can answer them. There is no subject field on the form, so a visitor cannot title a message "Re: your overdue invoice" in the owner's inbox. `From` is set by `ServerEmail.Send` from the domain owner's configured address and is on the `reservedHeaderNames` denylist, so nothing in a template can change it — the sending provider requires it too, which makes a drifting `From` a delivery failure rather than a cosmetic one.

The `read-form` step's schema is an allowlist: a field the schema does not declare is never read, and values are bounded, format-checked, and required there rather than trusted from the browser. `read-form` also reads the request body only, so a crafted link cannot append to a field through the query string.

## The `submit` action

`read-form` → `send-email` → `view-html`. The form posts with `hx-swap="outerHTML"` against itself, so `success.html` replaces it and a sent message cannot be resubmitted by pressing the button again.

`submit` and `view` declare the same access: `roles: ["author"]` plus `stateRoles: {"published": ["anonymous"]}`. An anonymous grant is still scoped by state, so a form that has been created but never saved accepts nothing, while the author can still exercise it. The form is submittable exactly when the page is viewable, which is pinned by `TestContactFormTemplate_ViewMatchesSubmit` in service/template_contactForm_test.go.

Access is granted to `anonymous` rather than to a `viewer` role because `model.NewStream` starts with no Groups — a sharing-gated contact form would be invisible to the public on the day it was created. Published means public here, which is why there is no `sharing` action.

## Two contracts that must not drift

The `send-email` step's `values:` must supply every key the `contact-form` email interpolates. `To`, `Subject`, and `ReplyEmail` are checked when Templates load, because the email's `to`, `subject`, and `headers` templates reject a missing key. The body's keys — `Name`, `Message`, `HeaderMessage` — are not, because `body.html` is `html/template` and renders a missing key as `""`. `TestContactFormTemplate_SuppliesEveryBodyKey` covers that gap.

Second, every visitor value the step reads with `.GetString` must first be declared to `read-form`, or it is silently never read. `TestContactFormTemplate_ReadFormDeclaresEveryVisitorField` pins that.

## No spam protection yet

This template ships with no bot defense of any kind — no honeypot, no rate limit, no timing check. That work is tracked separately in `emissary-specs/projects/FORM-SPAM-PREVENTION.md`, and the anti-abuse steps belong at the head of the `submit` pipeline. Do not deploy a public contact form until it lands.
