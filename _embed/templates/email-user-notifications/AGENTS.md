# email-user-notifications — Agent Notes

See [README.md](README.md) for what this email is. These are the non-obvious rules.

## Every value in a line is plain text from another server

`Name` and `Summary` come from a remote actor's profile and post. [body.html](body.html) prints them with plain `{{.Name}}` and `{{.Summary}}`, which html/template escapes. Never pipe them through `html`, `htmlMinimal`, `markdown`, or any helper that returns `template.HTML`. `TestUserNotificationsEmail_Render` sends markup through both and fails if any of it arrives unescaped.

## An encrypted message prints a label, never its content

`notificationEmailSummary` (service/domainEmail.go) returns "Encrypted message" for an MLS notification, whatever its stored summary holds. The template has no way to reach the notification itself, only the line built from it, so keep it that way: pass lines, not Notifications.

## The `to` and subject templates are `text/template`

A key that is missing from the data map prints the literal `<no value>` there. `TestUserNotificationsEmail_RequiredKeys` pins the keys those two templates read (`Email` and `CountLabel`); add a key to `SendUnreadNotifications` before using it in either one.

## This folder is not minified

Email bodies are parsed without the HTML minifier, so an HTML comment here is delivered to the recipient. Use `{{/* */}}` for comments.

## The email has no unsubscribe header, on purpose

The way out is the link to the Notification Settings page (emissary-specs NOTIFICATION-IMPROVEMENTS §1.6). There is no public unsubscribe route and no `List-Unsubscribe` header. Adding one needs a route that changes a User's settings without a sign-in, which is deferred (§3.2).
