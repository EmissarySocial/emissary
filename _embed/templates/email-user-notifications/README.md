# Unread Notifications Email

Tells a User that they have notifications they have not read. The hourly `SendNotificationEmails` task (consumer/sendNotificationEmails.go) calls `Notification.EmailUnread` (service/notification_email.go) for each User who holds an unread notification, and that method calls `DomainEmail.SendUnreadNotifications` (service/domainEmail.go) when the User is owed an email.

A User is owed an email when they hold an unread notification that is newer than their watermark (`notificationEmailDate`) and older than the span they chose on the Notification Settings page (`notificationEmailHours`). The email lists every unread notification they hold, newest first, up to 60. Sending moves the watermark to the newest unread notification, so the same notifications never cause a second email.

[email.hjson](email.hjson) declares the `user-notifications` emailId, the `User` model guard, and the to/subject templates. [body.html](body.html) is a Go html/template rendered by `ServerEmail.Send` (service/serverEmail.go) with the shared funcMap.

The data map provides:

- **`Email`, `Name`, `UserID`:** the recipient, read from the database by the task.
- **`CountLabel`:** the number of unread notifications in words, such as "3 unread notifications". The subject and the first line print it.
- **`Lines`:** one entry per listed notification, each with `Name` (who acted), `Action` (what they did), `Summary`, and `Date`.
- **`MoreCount`:** the number of unread notifications past the first 60, or zero.
- **`NotificationsURL`:** the All tab, `/@me/notifications/all`. A signed-out reader is sent to sign in and then back to it.
- **`SettingsURL`:** the Notification Settings page, which is the only way to change or stop these emails.
- **`Domain_Owner`, `Domain_URL`, `Domain_Name`, `Domain_Icon`:** added by `DomainEmail.Send`.

The rules that this email depends on are in [AGENTS.md](AGENTS.md).
