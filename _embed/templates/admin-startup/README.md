# Startup

The first-run setup checklist for a new domain. Unlike the other admin templates it is not served through `/admin/...`: `handler/startup.go` registers `/startup` and `/startup/:action`, calls `Template.LoadAdmin("startup")` directly, defaults the action to `page`, and permanently redirects to `/` once the Domain leaves `DomainStateStartup`, so the checklist only exists while setup is incomplete. [template.hjson](template.hjson) declares `model: Domain`, and all actions are owner-only.

The `page` action renders [page.html](page.html) (with `Cache-Control: no-store`): one row for each of the current theme's `startupTasks`, linking to the task's `href`. A row is checked off in one of two ways:

- **Its `value` is recorded on the Domain.** The `startup-save-task` steps in admin-domain, admin-groups, admin-rules, admin-searchTags, and this template's own `content` and `groups` actions record it, and so can a step in a theme package's own templates. The step ignores a value that the current theme does not list.
- **Its `value` names a connection provider, such as `GEOCODE-TILES`, and that connection is active.** The connection editor halts its pipeline, so no step can follow it; the page reads the connection instead. Such a task links to `/admin/connections/edit?providerId=...`, which opens the editor as a modal over the checklist.

When the server has more than one theme, the page opens with a theme button that calls admin-domain's `themes` chooser. This template writes no `themeId` of its own; `theme-switch` in admin-domain stays the one writer, and it reloads the checklist under the new theme.

`domain` ([domain.html](domain.html)) draws the theme's settings form in a modal, with the same steps as admin-domain's `index`, and records the same task value, `/admin/domain`, so either one checks the row off. A theme opts in by linking its first task to `/startup/domain`; only Qwerty does today. Inside this template the form is the theme's `startupForm` when it defines one, and its `form` otherwise (`model.Theme.PropertyForm`), so a theme can lay the same fields out differently for the checklist. Qwerty's is three tabs. A `startupForm` is a second copy of the fields, and the theme keeps the two in step. The two step lists are copies, so a change to one upload step goes in both.

`content` ([content.html](content.html)) runs `startup-create-streams`, which creates the theme's `startupStreams` and sets the Domain's start pages from the theme's `startupDefaultPages`. `groups` ([groups.html](groups.html)) runs `startup-create-groups`, which creates the theme's `startupGroups`. `complete` runs `startup-complete` then forwards to `/@me`. Nothing requires a task to be done first: "(Optional)" in a task's label is the only marker. The banner partial in admin-common's startup.html points owners back here from every other admin page while startup mode lasts.

`TestStartupTasks_CanBeCompleted` in [tests/startup_test.go](../../../tests/startup_test.go) fails for any theme, here or in a sibling package, that lists a task nothing can check off.
