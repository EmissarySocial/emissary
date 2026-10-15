# Previous/Next Buttons

This widget (widgetId `prev-next`) shows two buttons that link to the current stream's previous and next siblings, like the page footer of a documentation site. It is usually placed in an article's BOTTOM slot.

[widget.hjson](widget.hjson) has five settings: `sort` picks the order siblings are read in (`rank`, the site map's order, by default; or `label` or `publishDate`); `showLabels` adds a "Previous" or "Next" label above each title; `showSummary` and `showIcon` add those parts to each button; and `includeFamily` lets Previous fall back to the parent when there is no previous sibling, and Next to the first child when there is no next sibling. Every button always shows its title. A button whose stream has no summary or icon simply omits that line.

`includeFamily` is on by default, and the template reads it as a string, not with `GetBool`: a widget saved before the option existed has no value for it, and `GetString` returns `""` for that and `"false"` only when it was turned off. `GetBool` would read both as off. The other toggles are off by default, so `GetBool` reads them correctly.

[widget.html](widget.html) asks the Stream builder for `.PrevSibling` and `.NextSibling` with the chosen sort field and the `view` action. Both apply the visitor's permissions and the publish date, and return an empty builder when there is no neighbor, which `.IsEmpty` detects. With no neighbor on either side, the widget renders nothing; with only one, that one button renders, carrying the `previous` or `next` class so a theme can keep it on its own side.

## Classes, for themes

| Class | On |
| --- | --- |
| `widget-prev-next` | The wrapper, a `<div role="navigation">` |
| `prev-next-link`, plus `previous` or `next`, plus `sibling`, `parent`, or `child` | Each button, an `<a>` with `rel="prev"` or `rel="next"` |
| `prev-next-direction` | The optional "Previous" or "Next" label |
| `prev-next-heading` | The arrow, the optional icon, and the title. The arrow leads on Previous and trails on Next |
| `prev-next-arrow` | The `‹` or `›`, hidden from screen readers; the link's `rel` carries the direction |
| `prev-next-title` | The neighbor's label, always shown |
| `prev-next-icon`, `prev-next-summary` | The optional parts |

`theme-global`'s [03-widgets-prev-next.css](../theme-global/stylesheet/03-widgets-prev-next.css) gives the default look. A theme sets the space between the two buttons with `--prev-next-gap` on `.widget-prev-next`; each button's width follows it, so the gap stays exact.

## Things that look removable and are not

- **The wrapper is not a `<nav>`.** `theme-default`'s layout rules, which most themes copy, style every `nav` and `nav a` as the site's top bar.
- **One error cuts the widget off mid-tag.** Go templates cannot catch an error partway through, and `Stream.Widgets` keeps whatever was written before it, which can unbalance the whole page (BUG-279). Every call here must exist on the running builder; `Icon` needs a server built after `build.Stream.Icon()` was added.
- **Siblings with equal sort values are skipped.** `PrevSibling` and `NextSibling` look for a value strictly less or greater than this stream's, so two siblings with the same rank never link to each other. Give siblings distinct ranks, or sort by `label`.
