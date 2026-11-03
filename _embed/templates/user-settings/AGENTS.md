# user-settings — Notes for AI Agents

See [README.md](README.md) for what this template does. Template rules that span folders are in [../AGENTS.md](../AGENTS.md).

## Push notifications are wired by a script, not the form

The `notifications-form` action draws [notifications-push.html](notifications-push.html) before its `edit` form, and that script connects the form's `push-device-toggle` to the browser's Web Push subscription.

- **The toggle's id must stay an `id:` field.** The script finds `id="push-device-toggle"`. As a `path:`, the toggle widget makes its own id (`toggle-<path>-<value>`), and the wiring silently stops finding it.
- **The script polls for the toggle.** Its `view-html` runs before the form exists, so it waits for `role="switch"` and the toggle's hidden input, or an early state change is lost.
- **The toggle is browser state, applied on Save.** It is not a schema field. The script sets it from the real subscription and acts on Save, because `Notification.requestPermission()` needs that click. Save is an htmx request, so the script listens for `htmx:beforeRequest` from the toggle's form.
- **Send the toggle's events through hyperscript.** The widget's state changes on `setTrue` and `setFalse`, fired with `_hyperscript.internals.runtime.triggerEvent`, because `on` handlers do not reliably catch a plain `dispatchEvent`.
- **409 retries once, 403 never.** A 409 means the browser's endpoint belongs to another user, so the script unsubscribes and subscribes once more. A 403 means the server refused the endpoint (`WebPush.EndpointIsAllowed`), and subscribing again would destroy a good subscription.
