# Password Reset Confirmation Email

Confirms to a User that their password was just changed through the reset-code flow, and tells them to contact support if they did not do it. It is sent from `PostResetCode` (handler/signin.go) via `DomainEmail.SendPasswordResetConfirmation` (service/domainEmail.go) right after the new password is saved and the reset code is cleared; a send failure is only logged, so it never blocks the reset itself.

[email.hjson](email.hjson) declares the `user-password-reset-confirmation` emailId, the `User` model guard, and the to/subject templates; [body.html](body.html) is a Go html/template rendered by `ServerEmail.Send` (service/serverEmail.go). The data map provides `UserID`, `Username`, `Name`, `Email`, `ResetCode`, `ExpireDate` (both already zeroed at this point, since the code is cleared before sending), plus `Domain_Owner`, `Domain_URL`, `Domain_Name`, and `Domain_Icon`; the body currently uses only `Name`, `Domain_Name`, and `Domain_Icon`.
