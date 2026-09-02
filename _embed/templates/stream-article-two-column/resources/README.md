# Resources

Static files for [stream-article-two-column](../), served at `/.templates/article-two-column/resources/{filename}`. The folder needs no declaration in [template.hjson](../template.hjson) — `service.Template.Add` picks up a `resources` directory wherever it finds one.

The three SVGs are the Column Split options, one per value of the `data.columns` enum, and they are named for it (`columns-TWO-THIRDS.svg`, `columns-ONE-HALF.svg`, `columns-ONE-THIRD.svg`) so a template can address one by the stored value. Each draws two rounded rectangles in the proportion its name gives to the **left** column, on a 16×16 viewBox that lines up with the Bootstrap Icons used everywhere else in the UI.

## They cannot take their color through an `<img>`

Each icon is `fill="none" stroke="currentColor"`, so it paints in whatever `color` its surroundings have — but only where `currentColor` has a value to read. Referencing one through `<img src>`, `background-image`, or `content:` loads it as a separate document that inherits nothing from the page, and `currentColor` falls back to black. There is no error, and against a light background it looks close enough to correct to ship.

The way out is `mask-image` with `background-color`, which is what both pickers use: a mask takes only the alpha channel of the file, so the color comes from the element rather than from the SVG. See `.two-column-split-picker` in [stylesheet/two-column.css](../stylesheet/two-column.css). Inlining the markup — copied into a template, or reached through `<use>` from an inline `<symbol>` in the same document — is the other option, and the only one if an icon ever needs more than a single flat colour.
