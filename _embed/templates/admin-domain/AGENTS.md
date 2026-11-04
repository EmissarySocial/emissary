# admin-domain — Notes for AI Agents

See [README.md](README.md) for what this template does. Template rules that span folders are in [../AGENTS.md](../AGENTS.md).

## The theme is changed only by `theme-switch`, and its rule is written twice

`themes` and `theme-confirm` only draw views. `theme-switch` is the one action here that writes `themeId`, and it saves only when its `if` condition passes: the requested ID names a visible installed theme (one of `.Themes`) and is not the current one. [theme-confirm.html](theme-confirm.html) repeats the same test to decide whether to offer the Change Theme button. The two must agree, and `TestAdminDomainThemes_ConfirmMatchesSwitch` in [service/template_adminDomainThemes_test.go](../../../service/template_adminDomainThemes_test.go) pins it.

The check is the only one. The schema bounds `themeId` by length alone, and `GetTheme` returns hidden themes such as `global` and falls back to the default theme for an unknown ID, so without it a crafted link stores a theme that no owner can see or choose.

## "Current" is `(.Theme .ThemeID).ThemeID`, never `.ThemeID`

A Domain that has never chosen a theme stores `""` and runs the default theme. Compared against the stored value, the default theme would be offered as a change from itself.

## `theme-switch` stays outside `as-modal`, and refuses by redrawing `theme-confirm`

`as-modal` adds `closeModal` to every POST, so a refusal inside it closes the modal and takes its message along. `inline-error` is no better: it targets `#htmx-response-message`, and the settings form behind the modal carries one earlier in the document, so the message lands behind the modal. The Change Theme button therefore targets `#modal-window`, and the refusal branch renders [theme-confirm.html](theme-confirm.html), which by then explains why. A refusal must not become an error either, because `errorHandler` writes every error to the ErrorLog.

## The theme is chosen on this page, never in a theme's form

`Domain.PropertyForm()` renders the current theme's own `form`. A theme field placed there would have to be repeated in every theme, and a theme that left it out would strand the owner on it. One POST would also save the old theme's settings and switch away from them. The Theme card in [index.html](index.html) sits outside that form for both reasons.

## The `onerror` handlers on theme images are load-bearing

Qwerty and the emissary-dev theme ship no `resources/startup/icon.png` or `splash.png`, so both are a 404. The inline handler in [theme-icon.html](theme-icon.html) swaps in a palette icon, and the one in [theme-confirm.html](theme-confirm.html) hides the splash image. A hyperscript `on error` may attach after the error has already fired. Sonar flags these handlers as mouse or keyboard listeners (Web:S6847), which they are not.

## Two other writers of `themeId` skip the rule

- **`admin-startup`'s `choose-theme`** saves any `themeId` from the URL, while the Domain is in startup.
- **`admin-sso`'s own `themes` action** saves `themeId` from a form, with no check. Nothing links to it, and its [themes.html](../admin-sso/themes.html) posts to `/admin/domain/themes`, which saves nothing.
