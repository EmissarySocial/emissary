# theme-global — Agent Notes

See [README.md](README.md) for what the theme is and what each numbered stylesheet covers. These are the rules behind it that look removable and are not.

## Every dark-mode gray is hue 0, and a one-digit slip is invisible until it is ugly

The gray ramp in [stylesheet/01-colors.css](stylesheet/01-colors.css) is IBM's **warm gray**, not IBM's neutral gray, and the dark block inverts it. Warm gray is defined by red sitting above green and blue while **green and blue stay equal** — hue 0 in HSL. That equality is the whole rule: it is the only way a 17-step ramp reads as one family, and at these saturations (2% to 7%) it is also the only property a reader can perceive. Brightness differences look intentional; hue differences look like dirt.

A wrong digit in the green channel does not break the ramp visibly on its own step. It breaks the *transition* to the steps beside it, and only where two of them meet on screen. `#171714` instead of `#171414` moves gray00 from hue 0 to hue 60, which is olive, and at 8% lightness nothing about it reads as "green" in isolation; it reads as a smudge next to gray05. Six steps drifted this way before anyone could name what was wrong with the backgrounds, and gray60 had separately been filled from the neutral ramp (`#a8a8a8`, the light-mode gray40) instead of warm-gray-40, putting a colorless hole in the middle of a warm scale.

So: when you add or edit a dark gray, check `G == B` before checking anything else, and take new values from IBM warm gray rather than IBM gray. Eight of the steps are IBM warm gray exactly (`#ada8a8`, `#8f8b8b`, `#726e6e`, `#565151`, `#3c3838`, `#272525`, `#171414` and the three light steps rounded to hue 0), and the in-between steps (gray01 through gray05, gray15) are interpolations along the same line. Light mode is the opposite convention and is correct as it stands: it is IBM's **neutral** gray, every step fully desaturated, and warming it would be a separate design decision rather than a fix.

Nothing catches a violation. CSS has no notion of a palette, the ramp is not covered by a test, and the defect only shows up as an unpleasant page that no one can quite diagnose.

## `--white` and `--black` are pure white and pure black in both schemes

`--white` is always `#ffffff` and `--black` always `#000000`; the dark block does not redefine either. Use them only for color on something that keeps its color in both schemes: a photo, a map, or an accent that stays saturated in dark mode. The page's own surfaces and the text on them use `--gray00` and `--gray100`, which flip.

- **The color ramps flip unevenly.** In dark mode the -70 steps turn light, `--blue50` and `--red50` turn darker, and `--blue60` stays the same, so check a background's dark value before choosing `--white` or `--gray00` for the text on it.
- **The dark ramp's top is `#fffafa`, not pure white.** Pure white is the only color at L\* 100, so carrying the ramp's warmth all the way up costs a little brightness. `#fffafa` spends 1.4 L\* to land at chroma 1.76, matching `--gray90`'s 1.77 exactly, and the contrast against the page only moves from 18.3:1 to 17.7:1.
- **`--button-primary-color` stays a literal `#ffffff`.** Some themes outside Emissary still redefine `--white` per scheme, and `var(--white)` there would turn the label dark on a saturated button.

## The extended palette is a second, unrelated ramp

[stylesheet/01-more-colors.css](stylesheet/01-more-colors.css) defines `--color-gray-50`…`--color-gray-900` in a Spectrum-style scale, and those grays are **neutral in both schemes**. They are not the same family as `--gray00`…`--gray100` and do not follow the warm-gray rule above. Mixing the two ramps in one component is what makes a panel look subtly cold against its own page; pick one ramp per surface.

## Every file in `javascript/` ends with a semicolon

The `javascript` bundle joins its files with nothing between them, and the minifier then reads the result as one program. A file ending in `})()` followed by one starting with `(function` becomes `…})()(function…`, a call on the first file's return value: it throws, and every script after it in the bundle (turboclick, for one) silently never runs. `prism.js` was first written this way during EMISSARY-DEV-WEBSITE §1.7, and only running the minified bundle caught it. [service/theme_javascript_bundle_test.go](../../../service/theme_javascript_bundle_test.go) checks every bundled javascript file under `_embed/templates`.

## A `def` that waits makes its caller wait, and an `on` handler queues behind it

Hyperscript's `call` awaits an async `def`, and a `def` becomes async as soon as it contains `wait`. An `on click` handler that calls one stays busy until the wait ends, and hyperscript queues every later click on that element behind it, so a second click looks ignored for seconds. 0.9.93 has no `async` command to detach the wait. [hyperscript/highlight._hs](hyperscript/highlight._hs) avoids waiting at all: its animation runs once and ends on the normal background, so `.flash-highlight` can stay on the element, and removing and re-adding it (with a layout read between) replays the flash.

Two more traps found on the way. `wait for animationend or 3s from node` parses without error but never times out; the timeout works only with an argument list, as in `wait for animationend(animationName) or 3s from node`. And `02-accessibility.css` shortens every animation to 0.01ms under `prefers-reduced-motion`, so an animation that carries meaning needs its own reduced-motion rule: `.flash-highlight` uses a class selector with `!important`, which outranks the universal one, and a `step-end` timing that holds the color without fading.

