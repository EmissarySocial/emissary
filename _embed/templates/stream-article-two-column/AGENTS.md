# stream-article-two-column — Agent Notes

See [README.md](README.md) for what this Template is and how the split is set. These are the rules behind it that look removable and are not.

## The radios' document order IS the ladder

`data.columns` has five values, and the chevrons in [editor.html](editor.html) reach the next one with `previousElementSibling` / `nextElementSibling`. There is no lookup table anywhere: the markup order is the only statement of which split is "one step narrower". So the radios are emitted narrowest-left first (`ONE-QUARTER` through `THREE-QUARTERS`), and reordering them silently reverses the control — which is the exact defect the chevrons replaced, because the icon row they replaced was ordered the other way. `TestTwoColumn_EditorLadderOrder` is the only thing that catches it.

The `.two-column-split-values` wrapper exists for that walk. It is `display: contents`, so it adds nothing to the flex row, but it guarantees the five radios are contiguous siblings with nothing interleaved — which is what makes the walk, and the `:first-child` / `:last-child` that fade the end chevrons, correct rather than merely true today. Unwrap it and the walk starts finding buttons.

## Stepper rules hang off `.two-column-split-stepper`, never off the shared class

`.two-column-split-picker` is the chrome both surfaces share; the edit page adds `.two-column-split-stepper` for everything that mirrors the columns. The distinction is not tidiness. A `gap` put on the shared class to space the stepper's rails by a gutter's width also reaches the Layout tab, where it blows the five icons 32px apart — which shipped, and read as a spacing bug with no obvious cause. `--two-column-gap` is declared on the stepper for the same reason.

## The control tracks the dividing line by riding a mirror of the columns

The edit page's control is two chevrons and nothing else — no icon between them, by decision — so the five SVGs serve the Layout tab alone, and the dividing line falls in the 4px gap the chevrons straddle. The picker is a two-item flex row — `.two-column-split-rail-left` and `-right` — carrying the same `flex-basis: 0` and `flex-grow` values as the editors below it, from the same `:has(:checked)` trigger. The rails draw nothing; they exist so their shared boundary lands where the gutter does. `.two-column-split-control` then sits inside the **left** rail at `left: calc(100% + var(--two-column-gap) / 2)` with `translateX(-50%)`, which centers it on the gutter's midpoint, and that is exact at every stop rather than approximate: the left rail's width *is* the left column's width, and the half-gutter offset is read from the same variable the columns space themselves with. Lift the control out of the left rail and it silently stops tracking while every other thing on the page keeps working.

Only the **left** rail is positioned, and the control carries a `z-index`. Both halves of that matter: the control's right half overhangs the right rail, so a positioned right rail paints after the left rail's entire subtree and swallows every click on the right chevron. That shipped too — the divider would travel left and then stick, with nothing logged. `TestTwoColumn_ControlStaysClickable` guards it.

`--two-column-gap` therefore has two consumers that must agree, which is why it is declared once on `.two-column, .two-column-split-stepper` rather than inline. Below 768px the rail ladder does not apply, so the rails stay equal and the control sits centered — correct, because the columns are stacked and there is no dividing line to sit on.

## `twoColumnSplitStep` finds the radios document-wide, on purpose

It queries `<input[name='data.columns']:checked/>` instead of reaching up to the picker it was clicked in, because an inline hyperscript query literal cannot start with a class — the minifier escapes the `<` and the whole block stops parsing, silently. The repo's root `AGENTS.md` has that rule; `TestTwoColumn_EditorLadderOrder` fails on any `&lt;` in the rendered block. The cost is the assumption that one page holds one such group, which holds here: the edit page has a single picker, and the Layout tab's copy is a separate document.

## Assigning `checked` fires no `change` event

Only user interaction does. `twoColumnSplitStep` therefore sends one itself, and that send is what saves the article: hyperscript's `send` builds an `Event` with `bubbles: true`, so it reaches the `on change` handler on the picker. Drop the send and the chevrons still move the divider, still re-proportion the editors, and never persist anything.

## The chevrons are mouse affordances, not the control

The radiogroup is the keyboard and screen reader path: Tab lands on the checked radio, arrow keys walk the ladder natively, and the focus ring is drawn on `.two-column-split-control` because the radio itself is invisible. On the edit page that ring is the *only* focus affordance, so a rule that stops matching it takes keyboard focus off the screen entirely and nothing reports that. The group carries its own `aria-label`, since the edit page has no visible caption to point at — the Layout tab's copy still has one. That is why the hidden radios stay focusable, and why the stylesheet parks them inside the picker instead of off-screen — focusing an off-screen radio scrolls the page to nothing. The chevrons carry `tabindex="-1"` and `aria-hidden="true"` so assistive tech hears one control instead of three.

One consequence is visible and accepted: a native radiogroup **wraps** at the ends, so arrow keys cycle from `THREE-QUARTERS` back to `ONE-QUARTER` while the chevrons stop. Suppressing it would take a `keydown` handler that preventDefaults at both ends.

The ends of the range are CSS alone — `:has(input:first-child:checked)` fades the left chevron and sets `pointer-events: none`. There is deliberately no `disabled` attribute, because an attribute would need syncing at render *and* after every step, and two implementations of one rule drift. The `if no target` guards in `twoColumnSplitStep` are what hold if `:has()` ever does not.

## Saving goes through `save()`, debounced, and both halves matter

`save()` calls `beforeSave()`, which copies each CodeMirror back into the `<textarea>` that actually posts; submitting `#saveForm` directly posts whatever the textareas last held and silently reverts typing. The debounce is not tidiness either: `save()` takes a lock, and a second call arriving mid-flight waits for the first and then exits **without saving**, so a quick run of chevron clicks would leave the last one unsaved. One debounced call reads the final state.

`beforeSave()` also has to be defined after [stream-article-base/edit-menubar.html](../stream-article-base/edit-menubar.html) renders its no-op of the same name, since a later definition wins, and its body is a plain JavaScript call because it must behave identically whether or not EasyMDE loaded.

## The editor row carries no `columns-*` class

The page reads a stored class; both pickers read `:has(:checked)`. That is the whole reason the control sits where the writing happens — a server-rendered class still says whatever was last saved, and the writing surface is supposed to re-proportion on the click. It also means the picker must stay a **preceding sibling** of `.two-column-editor`, because the ratio rules reach it with `~`. Move it inside the row or below it and the editors keep working and simply stop responding.

## `flex-basis: 0` is what makes `flex-grow` a ratio

`flex-grow` shares out only the space left over after each item takes its basis, and the default basis is the item's own content width. Without the zero basis a long left column and a short right one split just the remainder 2:1, and the proportion drifts with whatever the author wrote. It bends the layout rather than breaking it, which is the kind of wrong nobody reports.

The reflow breakpoint is 768px measured with `@container` against `.page`, matching every responsive utility in theme-global, and nothing here declares a `container-type` of its own. Declaring one closer in — on the article's own column — would put the two width settings in charge of each other, so a layout-MEDIUM article could never show an asymmetric split at all: 66% of the widest page the theme draws is still under 768px. The flip side is accepted: a narrowed article still splits on a wide page, exactly as every other `.cols-*` grid in the app does.

## Both surfaces must gain a rung together

[editor.html](editor.html) steps the ladder and [layout-controls.html](layout-controls.html) draws all of it, from one enum in [template.hjson](template.hjson) and one stylesheet. Adding a value means a radio in each file, a page rule, an editor rule and a rail rule in [stylesheet/two-column.css](stylesheet/two-column.css), a label rule, and an SVG in [resources/](resources) — see that folder's README for why those icons cannot be `<img>`. Miss the stylesheet and the new value saves fine and renders as equal halves.

Both files list the radios in the same ladder order, narrowest left column first, and `TestTwoColumn_LadderOrderOnBothSurfaces` holds them to it. On the edit page that order is executable — it is what the chevrons walk — and on the Layout tab it is what stops the icon row reading backwards, which is the defect that started all of this.

## Each radio is emitted whole, from inside its branch

Two traps make that the only safe form, and neither raises anything. Templates are minified before they are parsed, and an action inside a **tag** is read as attribute markup and lowercased, so `{{if eq $columns "TWO-THIRDS"}}checked` written between two attributes compares against `two-thirds` and never matches. Hoisting `checked` into a variable dodges that and hits `html/template` instead: a value interpolated where an attribute **name** belongs goes through `htmlNameFilter`, which replaces anything invalid — the empty string included — with the literal text `ZgotmplZ`, so every unchecked option gets a junk attribute. `service/template_minify_test.go` guards the first; `TestTwoColumn_EditorPickerChecksExactlyOne` guards the second.

`ONE-HALF` is the final `else` rather than a case of its own, so an unset or unrecognized value selects it. That is honest as well as convenient: an empty value renders `columns-` on the page, which matches no rule and leaves the stylesheet's equal-halves default in place. It also keeps one radio always checked, which is what makes the field present in every POST.

## This Template publishes an empty body, on purpose

Nothing here writes `content.*`, and `content.HTML` is the only body a remote reader ever sees. The two columns live in `data.left` and `data.right` because a Stream has one content area, and `set-data` cannot paper over it — `content.HTML` is produced by `service.Content.New`, which only the `edit-content` step calls. The Summary on the Info tab is what previews show. See the repo's root `AGENTS.md`.
