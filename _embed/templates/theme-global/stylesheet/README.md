# Emissary Global Design System

This directory holds the baseline stylesheet for every Emissary theme. Individual themes build on top of these files, so anything defined here is available everywhere. This document is an orientation for designers who will use and extend the system.

## How the files are organized

Files are named with a numeric prefix that controls cascade order — lower numbers load first, so later files can override earlier ones. The prefixes group into layers:

- **`00-` Reset** — [normalize.css](00-normalize.css) v8.0.1, unmodified. Cross-browser baseline.
- **`01-` Color tokens** — the raw color palettes as CSS custom properties.
- **`02-` Foundations** — typography, forms, accessibility, animations, backgrounds, content.
- **`03-` Widgets** — self-contained components (cards, modals, tabs, tooltips, menus, etc.). One file per widget.
- **`05-` Layout primitives** — margin, padding, positioning, geometry, scrolling.
- **`09-` Display & grid** — columns, flexbox, width, display utilities.
- **`10-` Environment** — responsive print/screen rules and view transitions.

The system is a hybrid: **semantic component styles** (a `.card` looks like a card on its own) combined with **atomic utility classes** (`.padding-lg`, `.flex-row`, `.text-sm`, `.cols-3`) that you compose in markup. Most layout work is done by stacking utilities; most "look" is carried by widgets and tokens.

## The rhythm unit

Almost every size in the system is a multiple of one variable, `--rhythm`, defined in [02-typography.css](02-typography.css):

```css
:root { --rhythm: 8px; }
```

Font sizes, line heights, margins, padding, border radii, and gaps are written as `calc(var(--rhythm) * n)` rather than as fixed pixels. This keeps everything on a consistent 8px vertical grid. When you extend the system, reach for `--rhythm` multiples instead of hard-coded pixels so your work stays aligned. The `.mobile` context bumps the rhythm to 10px for slightly larger touch spacing.

The spacing t-shirt scale used throughout margins and padding maps to rhythm multiples:

| Size | Value | Rhythm |
|------|-------|--------|
| `xs` | 4px | × 0.5 |
| `sm` | 8px | × 1 |
| `md` (default) | 16px | × 2 |
| `lg` | 32px | × 4 |
| `xl` | 48px | × 6 |

## Colors

There are two independent color systems. Know which one you are pulling from.

### The primary palette ([01-colors.css](01-colors.css))

Based on the IBM Design Language. Defines ramps for `--gray00`…`--gray100`, `--blue10`…`--blue100`, `--red*`, and `--green*` (steps of 10). On top of the raw ramps sit **semantic tokens** — this is what your markup and new components should reference, not the raw ramp values:

- `--body-background`, `--page-background`, `--page-border`
- `--input-background`, `--input-border`, `--input-color`, and their `-invalid` variants
- `--text-color`, `--heading-color`
- `--link-color`, `--link-color-hover` (hover is derived via `color-mix` toward black, so links darken consistently)
- `--focus-color` — the accent for focus rings, primary buttons, and interactive accents. It defaults to `--link-color`, but a page or container can override it inline to theme a whole region (for example, a conversation tinting its buttons and focus rings to its group color). Primary and outline buttons read `--focus-color` directly so they inherit these scoped overrides.
- `--border-radius` — defaults to one rhythm unit.

### The extended palette ([01-more-colors.css](01-more-colors.css))

A larger Spectrum-style set (`--color-gray-50`…`--color-gray-900` and 13-step ramps for blue, green, orange, red, celery, chartreuse, cyan, fuchsia, indigo, magenta, purple, seafoam, yellow). Use these when you need hues beyond the core four. Note the naming difference: dashes and a `100`–`1300` scale (`--color-purple-800`), versus the primary palette's `--purple`-free, no-dash `--blue60` style.

### Dark mode

Dark mode is handled entirely through `@media (prefers-color-scheme: dark)`. Rather than swapping every rule, the palettes **invert their own ramps** — in dark mode `--gray00` becomes near-black and `--gray100` becomes white, and `--white` itself flips to a near-black. Because components reference semantic tokens and ramp steps rather than literal hex, they adapt automatically.

There are two consequences to remember:

- A few values must **not** flip. `--button-primary-color` is a literal `#ffffff` (not `var(--white)`), because a primary button keeps a dark accent background in dark mode and its label must stay light.
- Font weight bumps up one step in dark mode (`--weight` 300 → 400) to hold legibility against dark backgrounds.

Dark mode is partly a work in progress — a block of semantic-token overrides in [01-colors.css](01-colors.css) is currently commented out. Treat the ramp inversion as the live mechanism.

Visibility helpers `.light-mode-show` / `.light-mode-hide` / `.dark-mode-show` / `.dark-mode-hide` let you show content in only one mode.

## Typography

Defined in [02-typography.css](02-typography.css).

- **Font stack** — the system UI sans-serif stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto…`). No web fonts are loaded. Monospace uses a matching system-mono stack via `.monospace` / `code`.
- **Base weight** is `--weight: 300` (light), and bolder styles are expressed relative to it: `.bold` is `--weight + 300`, `.extra-bold` is `+600`, `.text-thin` is `-100`. Deriving from `--weight` means the whole document re-weights correctly in dark mode.
- **Headings** `h1`–`h3` are sized in rhythm multiples with generous top margins; a heading that is `:first-child` loses its top margin.
- **Text-size utilities** run `.text-3xl` down to `.text-xs`. They apply to the element, and also to nested `p` and `input`, so you can size a whole block at once. `p` and `.text-md` are the default body size (16px / 24px line height).
- **Text-color utilities** — `.text-gray-00`…`.text-gray-90` map to the gray ramp; `.text-red`, `.text-green`, `.text-black`, `.text-white` map to semantic roles. `.text-gray` and `.text-light-gray` dim via opacity rather than a color swap, so they work on any background. `.text-nocolor` resets to `--text-color`.
- **Decoration & transform** — `.text-underline`, `.text-plain`, `.line-through`, `.text-uppercase`, `.text-lowercase`, `.text-capitalize`, `.italics`.
- **Truncation** — `.ellipsis` (single-line clip) and `.ellipsis-block`. `.nowrap` / `.wrap` control wrapping.
- **Behavior helpers** — `.noselect`, `.nopointer`, `.clickable` (also applied automatically to `[role="link"]` and `[role="button"]`).

## Forms and buttons

The largest foundation file is [02-forms.css](02-forms.css).

### Inputs

Text inputs, `textarea`, `select`, and anything with `.input` or `[role="textbox"]` share one look: full-width, rhythm-based padding, rounded corners, `--input-*` tokens. Focus draws a 2px outline in `--focus-color`. Invalid controls (`:invalid`, `.invalid`) draw a **red** outline instead — the red rule is placed after the accent rule on purpose so source order breaks the specificity tie. When you build a fake-input wrapper (`.input` / `[role="textbox"]` around a real control), the outline is drawn on the wrapper and suppressed on the inner control, so you never get a double ring.

### Rich controls

- **`.checkbutton` / `.radiobutton`** — the native box is hidden and the whole labeled block becomes the target, highlighting in blue with `:has(:checked)`.
- **`.toggle`** — an iOS-style switch driven by a `[value="true"]` attribute on `.toggle-container`; the marker slides via a transform.
- **`.multiselect`** — a bordered scrolling option list with a side button column.

### Buttons

`button` and `.button` are interchangeable — the class exists so a link or div can look like a button. The base is a neutral gray pill. Variant classes layer on intent:

- **`.primary`** — filled with `--focus-color` (so it picks up scoped accent overrides).
- **`.outline`** — transparent with an accent border.
- **`.warning`** — red. **`.success`** — green. **`.highlight`** / **`.selected`** — gray emphasis.
- **`.text-red`** — neutral until hover, then turns red (for destructive actions).
- **`.text-sm`** / **`.text-xs`** shrink the button's padding and radius to match smaller text.
- **`[disabled]`**, `.inactive` — muted, non-interactive.
- **`.barberpole`** — an animated diagonal-stripe progress state (there is a blue "working" version and a gray idle version).
- **`.button-group`** — joins adjacent buttons into a single segmented control with shared borders and rounded ends.

### Form layout

`.layout-vertical` stacks label-over-input rows (`.layout-element`), and `.layout-horizontal` lays them out in a flex row. `.layout-title`, `.layout-description`, and `.layout-heading` provide the standard form headings.

### HTMX integration

`.htmx-request-show` / `.htmx-request-hide` toggle content while an HTMX request is in flight, and `.spin` (plus the `spin` keyframes) drive loading spinners. Emissary is an HTMX-driven app, so these appear frequently.

## Widgets

Each `03-` file is one component. Reach for these before building your own:

- **[Card](03-widgets-card.css)** (`.card`) — bordered, subtly shadowed container; `.clickable` adds a hover border, `.selected` a heavy border. It is a container query context, so children can respond to the card's own width.
- **[Modal](03-widgets-modal.css)** (`#modal`) — a full-screen centered dialog with a blurred underlay. Supports fixed `#modal-header` / `#modal-body` / `#modal-footer` regions and `.large` / `.huge` sizes. Animates in on the `.ready` class.
- **[Tabs](03-widgets-tabs.css)** — driven by ARIA (`[role="tablist"]` / `[role="tab"]` / `[aria-selected]`). Five visual styles via a modifier class: default folder tabs, `.underlined`, `.pills`, `.wizard` (numbered stepper with done/disabled states), and `.vertical`.
- **[Tables](03-widgets-tables.css)** — `table.table` for read views, `table.grid` for inline-editable grids, and `div.table` for flexbox row layouts that behave like tables. Clickable rows use `[role=link]` / `.clickable`.
- **[Alerts](03-widgets-alert.css)** — `.alert-red`, `.alert-blue`, `.alert-green`, `.alert-gray` banner messages.
- **[Tags](03-widgets-tags.css)** — `.tag` pills (`.blue`, `.green`, `.warning` variants) and the `.info` callout box.
- **[Menu](03-widgets-menu.css)** — `.menu` with `[role=menuitem]` children; selection highlights with the primary accent.
- **[Pop-up](03-widgets-popUp.css)** (`.popUp` / `.popUp-content`) — a positioned floating panel that scales in when the container gets `.visible`.
- **[Tooltip](03-widgets-tooltips.css)** — `.tooltip` inside a `.tooltip-container`; appears above on hover/focus with a pointer triangle.
- **[Hover utilities](03-widgets-hover.css)** — `.hover-show` / `.hover-hide` / `.hover-swell` and a `.hover-trigger` parent that reveals or animates children on hover. All gated behind `@media (hover: hover)` so touch devices are unaffected.
- **[Autocomplete](03-widget-autocomplete.css)**, **[Picture](03-widgets-picture.css)**, **[Slideshow](03-widgets-slideshow.css)** (scroll-snapping carousel), **[Uploader](03-widgets-uploader.css)** (drag-and-drop file target), **[Selection](03-widget-selection.css)** (`.selected-show` / `.selected-hide`), **[Draggable](03-widgets-draggable.css)**.

## Layout and spacing utilities

These are the atomic classes you compose most often.

- **Margin** ([05-margins.css](05-margins.css)) and **padding** ([05-padding.css](05-padding.css)) — `.margin-{xs,sm,md,lg,xl}` and `.padding-*` on the rhythm scale, with directional variants (`-vertical`, `-horizontal`, `-top`, `-bottom`, `-left`, `-right`), `-none`, and `-auto`. The bare `.margin` / `.padding` equal the `md` size. All are `!important` so they reliably win over component defaults. Padding utilities have responsive `sm:` / `md:` / `lg:` prefixes.
- **Positioning** ([05-positioning.css](05-positioning.css)) — `.pos-absolute`, `.pos-fixed`, `.pos-relative`, `.pos-sticky`, plus corner-anchored helpers like `.pos-absolute-top-right` (inset by one rhythm) and `-0` variants (flush to the edge).
- **Geometry** ([05-geometry.css](05-geometry.css)) — aspect ratios (`.aspect-square`, `.aspect-16-9`, `.aspect-4-3`, `.aspect-2-3`), shapes (`.square`, `.circle`, `.rectangle` — all with `object-fit: cover` and a placeholder gray background), and corner rounding (`.rounded`, `.rounded-top`, `.rounded-bottom`). `hr` is styled here as a light rule with rhythm margins.
- **Scrolling** ([05-scrolling.css](05-scrolling.css)) — `.scroll-vertical`, `.scroll-horizontal`, `.scroll-both`, `.scroll-none`, all with `overscroll-behavior: none` to trap scroll chaining.

## Grids, flexbox, and width

- **Columns** ([09-columns.css](09-columns.css)) — the responsive tile grid. `.cols-1` through `.cols-6` name the **desktop** column count; the class automatically steps **down** to fewer columns on narrower screens (a `.cols-6` is 6-up on a wide desktop, 1-up on a phone). Widths are computed with `calc()` from a `--cols` and `--gap` custom property that the breakpoints reset. `.no-gap` removes gutters; a child `.col-2x` spans two columns. **These are container queries, not viewport queries** — put `.container` on a parent to establish the query context, so a grid reflows based on the space it actually occupies (essential inside cards, sidebars, and modals).
- **Flexbox** ([09-flexbox.css](09-flexbox.css)) — a full atomic set: `.flex-row`, `.flex-column` (and `-reverse`), `.flex-align-*`, `.flex-justify-*`, `.flex-grow-{0..4}`, `.flex-shrink-{0,1,2}`, `.flex-wrap`, and the shortcut `.flex-center`. Rows get a `--rhythm` gap by default. Every utility has `sm:` / `md:` / `lg:` container-query variants.
- **Width** ([09-width.css](09-width.css)) — fractional widths expressed multiple ways for convenience (`.width-1-2`, `.width-2-4`, `.width-50\%` all mean 50%), plus a `.max-width-*` pixel scale (16, 24, 32, 48, 64, 96, 128, 192, 240, 256, 320, 480, 512, 640, 800) with responsive prefixes.
- **Display** ([09-display.css](09-display.css)) — `.show` / `.hide`, `.block` / `.inline-block` / `.inline`, `.visible` / `.invisible`, text and vertical alignment, and floats — all with `sm:` / `md:` / `lg:` container-query variants.

## Responsiveness

The system uses **two** responsive mechanisms, and the distinction matters:

- **Container queries** (`@container`) drive the grid, flexbox, width, scrolling, display, and geometry utilities. Their `sm:` / `md:` / `lg:` prefixes react to the width of the nearest ancestor marked `.container` (or `.card`), not the viewport. This lets the same component reflow correctly whether it sits in a full-width page or a narrow sidebar.
- **Media queries** (`@media`) handle genuinely global concerns: dark mode, `prefers-reduced-motion`, print vs. screen ([10-responsive.css](10-responsive.css): `.print-hide`, `.screen-show`, `.mobile-hide`, `.desktop-show`), and view transitions ([10-transitions.css](10-transitions.css)).

The breakpoint ladder is consistent across the codebase:

| Prefix | Min width | Device |
|--------|-----------|--------|
| (none) | 0 | phone / base |
| `sm:` | 640px | large phone |
| `md:` | 768px | tablet |
| `lg:` | 1024px | desktop |
| (columns only) | 1280px | wide desktop |

## Accessibility and motion

[02-accessibility.css](02-accessibility.css) provides `.hide-visual` for screen-reader-only content (visually clipped, still in the accessibility tree). [02-animations.css](02-animations.css) defines the shared keyframes (`fadeIn`, `fadeOut`, `zoomIn`, `spin`, `barberpole`, etc.). Both animation-heavy files respect `@media (prefers-reduced-motion: reduce)`, which globally collapses animation and transition durations — honor it in anything new you add. Interactive components are keyed off ARIA roles and attributes (`[role="tab"]`, `[aria-selected]`, `[role="menuitem"]`), so keeping markup semantic is what makes the styling work.

## Extending the system

- **Compose utilities first.** Most layouts are built by stacking atomic classes in markup, not by writing new CSS.
- **Reference tokens, not literals.** Use semantic tokens (`--text-color`, `--focus-color`, `--page-background`) and rhythm multiples so your work inherits dark mode and stays on the grid.
- **Respect the cascade order.** If you add a file, its numeric prefix determines when it loads. Overrides belong at a higher number than what they override.
- **Prefer container queries for component-local responsiveness** and reserve `@media` for truly global concerns.
- **Theme a region with `--focus-color`.** Setting it inline on a container re-accents that region's buttons and focus rings without touching component CSS.
