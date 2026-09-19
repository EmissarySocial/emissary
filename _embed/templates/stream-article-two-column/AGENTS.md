# stream-article-two-column — Agent Notes

See [README.md](README.md) for what this Template is and how the split is set. These are the rules behind it that look removable and are not.

## Each option is placed on the dividing line it produces

The edit page positions all five options absolutely along a row above the editors, each at the point where it would put the gutter, so clicking an icon jumps straight to that ratio. An option at fraction f of the row belongs at `f` of the width, plus half a gutter, less `f` gutters — which is where the `calc()` offsets in `.two-column-split-stops` come from, all read off `--two-column-gap` so they cannot drift from the columns they point at. `TestTwoColumn_EveryValueIsDrawn` requires a stop rule per enum value, because a value with no `left` of its own does not fail visibly: it lands on another option and hides it.

Two consequences are worth knowing. The stops are **not** evenly spaced, so 1/4 and 1/3 sit closest; at the narrowest width that can still be in column mode (a layout-SMALL article on a 768px page, so 384px) their 32px boxes overlap by about 3px and the later one in document order takes that sliver. Below 768px the columns stack, there is no line to point at, and the options fall back to the Layout tab's evenly spaced row.

Clicking is plain `<label for>`: the radio selects itself, fires a native `change`, and the picker's debounced handler saves it. There is deliberately no script in that path, which is why the label must stay its input's **next sibling** — every rule that gives an option a position or an icon is written `input[value=X] + label`, so a label that moves loses both at once and lands unstyled on top of a neighbour.

## The radios' document order is what the keyboard walks

Both files list the five values narrowest-left-first, and `TestTwoColumn_LadderOrderOnBothSurfaces` holds them to it. On the edit page that order is not merely cosmetic: the stylesheet lays the options out left to right in the same sequence, and a native radiogroup's arrow keys follow **document** order — so a reordered list leaves the arrow keys jumping around the row instead of stepping along it, while every icon still sits in the right place. On the Layout tab the order is what stops the icon row reading backwards, which is the defect that started all of this.

## Every rule that positions an option hangs off `.two-column-split-stops`

`.two-column-split-picker` is the chrome both surfaces share; the edit page adds `.two-column-split-stops`. The distinction is not tidiness. A `gap` put on the shared class to space the edit page's row also reaches the Layout tab, where it blows the five icons a gutter's width apart — which shipped once, and read as a spacing bug with no obvious cause. `--two-column-gap` is declared on the stops variant for the same reason.

## The radiogroup is the whole control

There is no separate mouse path: the five labels are the control, and the radios behind them are the keyboard and screen reader path. Tab lands on the checked radio, arrow keys walk the ladder natively, and the focus ring is drawn on the option's own `<label>` because the radio itself is invisible. That ring is the only focus affordance either surface has, so a rule that stops matching takes keyboard focus off the screen entirely and nothing reports it — `TestTwoColumn_FocusRingHasATarget` is what notices. It is also why the hidden radios stay focusable, and why the stylesheet parks them inside the picker rather than off-screen: focusing an off-screen radio scrolls the page to nothing.

One consequence is accepted rather than fixed: a native radiogroup **wraps**, so arrow keys cycle from `THREE-QUARTERS` straight back to `ONE-QUARTER`. Suppressing that would take a `keydown` handler that preventDefaults at both ends.

## Saving goes through `save()`, debounced, and both halves matter

`save()` calls `beforeSave()`, which copies each CodeMirror back into the `<textarea>` that actually posts; submitting `#saveForm` directly posts whatever the textareas last held and silently reverts typing. The debounce is not tidiness either: `save()` takes a lock, and a second call arriving mid-flight waits for the first and then exits **without saving**, so a quick run of clicks, or a held arrow key, would leave the last change unsaved. One debounced call reads the final state.

`beforeSave()` also has to be defined after [stream-article-base/edit-menubar.html](../stream-article-base/edit-menubar.html) renders its no-op of the same name, since a later definition wins, and its body is a plain JavaScript call because it must behave identically whether or not EasyMDE loaded.

## The editor row carries no `columns-*` class

The page reads a stored class; both pickers read `:has(:checked)`. That is the whole reason the control sits where the writing happens — a server-rendered class still says whatever was last saved, and the writing surface is supposed to re-proportion on the click. It also means the picker must stay a **preceding sibling** of `.two-column-editor`, because the ratio rules reach it with `~`. Move it inside the row or below it and the editors keep working and simply stop responding.

## `flex-basis: 0` is what makes `flex-grow` a ratio

`flex-grow` shares out only the space left over after each item takes its basis, and the default basis is the item's own content width. Without the zero basis a long left column and a short right one split just the remainder 2:1, and the proportion drifts with whatever the author wrote. It bends the layout rather than breaking it, which is the kind of wrong nobody reports.

The reflow breakpoint is 768px measured with `@container` against `.page`, matching every responsive utility in theme-global, and nothing here declares a `container-type` of its own. Declaring one closer in — on the article's own column — would put the two width settings in charge of each other, so a layout-MEDIUM article could never show an asymmetric split at all: 66% of the widest page the theme draws is still under 768px. The flip side is accepted: a narrowed article still splits on a wide page, exactly as every other `.cols-*` grid in the app does.

## Both surfaces must gain a rung together

[editor.html](editor.html) places the options on the lines they produce and [layout-controls.html](layout-controls.html) spaces them evenly, from one enum in [template.hjson](template.hjson) and one stylesheet. Adding a value means a radio in each file, a page rule, an editor rule, a stop position and an icon rule in [stylesheet/two-column.css](stylesheet/two-column.css), and an SVG in [resources/](resources) — see that folder's README for why those icons cannot be `<img>`. Miss the stylesheet and the new value saves fine and renders as equal halves.

## Each radio is emitted whole, from inside its branch

Two traps make that the only safe form, and neither raises anything. Templates are minified before they are parsed, and an action inside a **tag** is read as attribute markup and lowercased, so `{{if eq $columns "TWO-THIRDS"}}checked` written between two attributes compares against `two-thirds` and never matches. Hoisting `checked` into a variable dodges that and hits `html/template` instead: a value interpolated where an attribute **name** belongs goes through `htmlNameFilter`, which replaces anything invalid — the empty string included — with the literal text `ZgotmplZ`, so every unchecked option gets a junk attribute. `service/template_minify_test.go` guards the first; `TestTwoColumn_EditorPickerChecksExactlyOne` guards the second.

`ONE-HALF` is the final `else` rather than a case of its own, so an unset or unrecognized value selects it. That is honest as well as convenient: an empty value renders `columns-` on the page, which matches no rule and leaves the stylesheet's equal-halves default in place. It also keeps one radio always checked, which is what makes the field present in every POST.

## This Template publishes an empty body, on purpose

Nothing here writes `content.*`, and `content.HTML` is the only body a remote reader ever sees. The two columns live in `data.left` and `data.right` because a Stream has one content area, and `set-data` cannot paper over it — `content.HTML` is produced by `service.Content.New`, which only the `edit-content` step calls. The Summary on the Info tab is what previews show. See the repo's root `AGENTS.md`.
