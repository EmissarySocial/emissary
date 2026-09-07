# theme-minimal — Agent Notes

See [README.md](README.md) for what the theme is and how [navigation.html](navigation.html) is laid out. These are the rules behind that layout that look removable and are not. Each is pinned by a test in [service/theme_minimal_navigation_test.go](../../../service/theme_minimal_navigation_test.go) unless noted.

## Never restyle or remove `class="framed"` on the bar's inner `<div>`

`.framed` is the shared page-width class from [stylesheet/02-page.css](stylesheet/02-page.css) that also frames `user-outbox/*` and the setup console. Inside `<nav>` it is additionally what gives the bar its flex layout, its centering, and its 1100px frame, through the `nav .framed` rules in [stylesheet/01-layout.css](stylesheet/01-layout.css). Below 640px the bar is hidden by hiding `<nav>` wholesale, and nothing inside it is restyled. Removing this class to build a mobile layout is exactly what broke the first attempt at the mobile menu: the desktop bar lost its centering and its max-width at the same time.

## The bar and the sheet are two elements that render the same links twice

One element cannot be both a horizontal bar and a full-screen sheet without every rule in one layout undoing a rule in the other, on markup the bar shares with pages that are not navigation at all. So `<nav>` is desktop-only, `.nav-menu` is mobile-only, and both range over the same `$navigation` variable, which is read once because `.Navigation` runs a database query. The cost is duplicate links, and the next rule is where it is paid.

## Sheet links carry no `id` and not the `nav-item` class, and both omissions are load-bearing

[SelectNav](../theme-global/hyperscript/selectNav._hs) resolves the current section with `document.getElementById('nav-' + id)`, which returns the first match in the document rather than the visible one, then runs `take .selected from .nav-item` and removes `aria-current` from every `.nav-item`. A duplicate id would make it pick the wrong copy, and the class would make it strip the sheet's mark. So the bar is marked client-side by SelectNav and the sheet is marked server-side from `.NavigationID`. A class token is matched whole, so `nav-menu-item` is invisible to `.nav-item`.

## The mobile menu is a `<div role="navigation">`, never a second `<nav>`

The bar's rules are element-scoped (`nav {…}` and `nav a {…}` in [stylesheet/01-layout.css](stylesheet/01-layout.css)), so a second `<nav>` silently inherits `height:48px`, `overflow-x:auto` and the whole bar treatment. The role on the wrapper is also what keeps a navigation landmark on the page while the menu is closed: `<nav>` is `display:none` below 640px and the sheet is hidden until the checkbox is checked.

## The disclosure is a checkbox, not `<details>`, and it precedes the label and the sheet in the DOM

`:checked` is a selector, so the 640px media query can switch the whole mechanism off above that width. A `<details>` keeps its open state in a DOM attribute that no media query can reach. The CSS reaches the label with `+` and the sheet with `~`, which fixes the sibling order. The checkbox also carries `autocomplete="off"`, or a back-navigation restores the menu open over the page the visitor just returned to.

## `role="button"` stays on the checkbox and never moves to the `<label>`

The role enrolls the control in the [a11y extension](../theme-global/javascript/a11y.js), which supplies the Enter key a native checkbox lacks. A `<label>` sits at tabindex -1, so the role there would make the extension focus it too, and one control becomes two tab stops.

## The Escape branch sets `aria-expanded` itself

`set my.checked to false` does not fire `change`, so the `on change` handler that syncs `aria-expanded` never runs on Escape. The Escape branch of the script attribute sets the attribute explicitly, and that duplicate-looking clause must stay. This one is not pinned by a test; it only shows up in a browser.

## `turboclick` goes on the sheet's links and not on the `<label>`

On a touch device the browser only synthesizes `mousedown` once it has resolved the gesture as a tap rather than a scroll, so a drag through the sheet never navigates and the class is as safe here as on the bar. On the label, the synthetic click toggles the menu twice and it lands back where it started. If a case ever needs excluding, the mechanism is turboclick's `.folder-handle` guard, not dropping the class.

## The skip link is emitted before both layouts

`<nav>` is `display:none` below 640px and the sheet is hidden whenever the menu is closed, which is always on load. A skip link inside either one stops existing for exactly the visitors it is there for.

## `nav .framed` uses `safe center`, not `center`

`<nav>` is a scroll container (`overflow-x:auto`), and a centered flex track that overflows it spills past both edges. The overflow before the start edge is unreachable because `scrollLeft` cannot go negative, so a site with eight or ten top-level pages silently loses its first link or two between 640px and roughly 800px, with no symptom beyond the bar appearing to start mid-word. `safe` falls back to start alignment exactly when the content overflows. The two values are indistinguishable until the content overflows, so "it looks right" proves nothing; render the same page against a stylesheet patched to plain `center` to see the difference. Not pinned by a test.

## Above 640px `.nav-menu` is `display:none`, not `hide-visual`

`hide-visual` deliberately leaves an element focusable, which is right on a phone, where the visible `<label>` stands in for the checkbox, and wrong on desktop, where there is no label to see. With `hide-visual` the checkbox is a dead tab stop on every desktop page. Not pinned by a test.

## The control is `position:fixed`, and its z-index sits between the sheet and the 1000 tier

The control is the close button over a fixed full-screen sheet, so an absolutely positioned one would scroll away while the sheet stayed. The sheet is 900 and the control 901. `.floating-menu`, `.popUp-content` and `#modal` share 1000, so a modal still opens over everything, and `.floating-menu` would paint over the open sheet if the stylesheet did not hide it. Because `.floating-menu` sits outside `.nav-menu`, no sibling combinator reaches it and `body:has(.nav-menu-toggle:checked)` is the only selector that does. Not pinned by a test.

## The control's backdrop is opaque and uses `--gray10`, not `--white`

Nothing in the page reserves space for the control, by design, so it floats over whatever scrolls past, which on this theme is often a photograph, and the backdrop is the whole compensation. The gray scale inverts between light and dark mode in [theme-global/stylesheet/01-colors.css](../theme-global/stylesheet/01-colors.css), so `--gray10` is one step off the page in both schemes (`#f4f4f4` on a `#ffffff` page, `#272725` on a `#171714` one), whereas `--white` is the page's own colour in dark mode and left the control readable only by its shadow. Check a token's value in both blocks before using it for contrast; the names describe a role, not a brightness. The residual weak spot is `--body-background`, which is `var(--gray10)` in both schemes, so the shadow is all that carries the control anywhere it floats over body background instead of a `.page`. Not pinned by a test.

## The sheet transitions `visibility`, not `display`, and reduced motion must zero the delay too

`display` is not transitionable, and its modern replacement (`@starting-style` with `transition-behavior: allow-discrete`) animates the entry only in Firefox 154: on exit `display` flips straight to `none` with no transition running, so the sheet would fade in and then vanish. `visibility:hidden` removes the sheet from the accessibility tree and the tab order exactly as `display:none` did. The visibility delay is what holds the sheet on screen for the length of the fade-out, and the theme's global reduced-motion switch in [theme-global/stylesheet/02-accessibility.css](../theme-global/stylesheet/02-accessibility.css) overrides `transition-duration` only. Without the explicit `transition-delay: 0s` rule, closing the sheet under reduced motion hides it instantly and then leaves a full-screen, invisible, still hit-testable box swallowing every tap for 200ms. Not pinned by a test.

## The sheet sets `overscroll-behavior: contain`

Without it a swipe that reaches the end of the sheet chains through to the document behind it, which scrolls invisibly and leaves the visitor somewhere else when they close the menu. A real body scroll lock needs script (the modal behavior in [theme-global/hyperscript/modal._hs](../theme-global/hyperscript/modal._hs) writes `document.body.style.overflow`); `contain` stops the chaining without any, which keeps the menu working with scripting off. Not pinned by a test.
