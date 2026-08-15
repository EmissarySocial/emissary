# User Welcome Email

Welcomes a newly registered user and gives them the link that completes their signup. After a visitor submits a valid registration form, `PostRegister` (handler/registration.go) calls `DomainEmail.SendWelcome` (service/domainEmail.go), which mints a JWT from the RegistrationTxn claims and embeds it in the button linking to `{{.Domain_URL}}/register/complete?token={{.Token}}`, where the user creates their password. Unlike most emails here it returns errors instead of swallowing them, so a failed send aborts the registration flow.

[email.hjson](email.hjson) declares the `user-welcome` emailId, the `User` model guard, and the to/subject templates; [body.html](body.html) is a Go html/template rendered by `ServerEmail.Send` (service/serverEmail.go). The data map provides `Username`, `Name`, `Email`, `Token` (the registration JWT), plus `Domain_Owner`, `Domain_URL`, `Domain_Name`, and `Domain_Icon`.
