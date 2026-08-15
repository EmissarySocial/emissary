# Single Sign-On

Admin console page for the JWT-based single sign-on integration, served at `/admin/sso` under the "External > Single Sign On" sub-menu. [template.hjson](template.hjson) declares `model: Domain` (built with `build.NewDomain`, owner-only) and a schema storing two values in the Domain's data map: `data.sso_active` and `data.sso_secret` (the JWT signing key, deliberately `format: unsafe-any`).

The `index` action combines [index.html](index.html) — an info box linking to the SSO documentation — with an inline `edit` step whose form (an active toggle and a secret-key field) is defined in the hjson, followed by `save`, `inline-save-button`, and `reload-page`, so the page edits in place with no modal. The hjson also carries `themes` and `signup` actions mirroring admin-domain's, with [themes.html](themes.html) rendering a tabbed theme picker that posts back to `/admin/domain/themes`. Extends `admin-common` for the shared menubar partial.
