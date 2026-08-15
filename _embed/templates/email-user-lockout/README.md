# User Lockout Email

Warns an account owner that repeated failed sign-in attempts have temporarily locked their account. `User.NotifySigninLockout` (service/user.go) sends it inline on the sign-in path via `DomainEmail.SendUserLockout` (service/domainEmail.go), swallowing errors so a mail failure never blocks sign-in. By deliberate rule the lockout does NOT change the stored password and no reset code is issued (resetting on unauthenticated input was the original CWE-645 bug), so the body only offers a button to the normal `/signin/reset` flow.

[email.hjson](email.hjson) declares the `user-lockout` emailId, the `User` model guard, and the to/subject templates; [body.html](body.html) is a Go html/template rendered by `ServerEmail.Send` (service/serverEmail.go). The data map provides `UserID`, `Username`, `Name`, `Email`, plus `Domain_Owner`, `Domain_URL`, `Domain_Name`, and `Domain_Icon` — intentionally no `ResetCode` or `ExpireDate`, so keep it that way when editing.
