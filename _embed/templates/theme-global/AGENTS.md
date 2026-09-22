# theme-global — Agent Notes

See [README.md](README.md) for what the theme is and what each numbered stylesheet covers. These are the rules behind it that look removable and are not.

## Every dark-mode gray is hue 0, and a one-digit slip is invisible until it is ugly

The gray ramp in [stylesheet/01-colors.css](stylesheet/01-colors.css) is IBM's **warm gray**, not IBM's neutral gray, and the dark block inverts it. Warm gray is defined by red sitting above green and blue while **green and blue stay equal** — hue 0 in HSL. That equality is the whole rule: it is the only way a 17-step ramp reads as one family, and at these saturations (2% to 7%) it is also the only property a reader can perceive. Brightness differences look intentional; hue differences look like dirt.

A wrong digit in the green channel does not break the ramp visibly on its own step. It breaks the *transition* to the steps beside it, and only where two of them meet on screen. `#171714` instead of `#171414` moves gray00 from hue 0 to hue 60, which is olive, and at 8% lightness nothing about it reads as "green" in isolation; it reads as a smudge next to gray05. Six steps drifted this way before anyone could name what was wrong with the backgrounds, and gray60 had separately been filled from the neutral ramp (`#a8a8a8`, the light-mode gray40) instead of warm-gray-40, putting a colorless hole in the middle of a warm scale.

So: when you add or edit a dark gray, check `G == B` before checking anything else, and take new values from IBM warm gray rather than IBM gray. Eight of the steps are IBM warm gray exactly (`#ada8a8`, `#8f8b8b`, `#726e6e`, `#565151`, `#3c3838`, `#272525`, `#171414` and the three light steps rounded to hue 0), and the in-between steps (gray01 through gray05, gray15) are interpolations along the same line. Light mode is the opposite convention and is correct as it stands: it is IBM's **neutral** gray, every step fully desaturated, and warming it would be a separate design decision rather than a fix.

Nothing catches a violation. CSS has no notion of a palette, the ramp is not covered by a test, and the defect only shows up as an unpleasant page that no one can quite diagnose.

## `--white` and `--black` are roles, not brightnesses

Both flip in the dark block: `--white` becomes the near-black page color (and must stay equal to `--gray00`, which it shadows) and `--black` becomes `#fffafa`, the warm white at the top of the ramp (and must stay equal to `--gray100`). Neither endpoint is a pure `#ffffff` or `#000000` in dark mode: pure white is the only color at L\* 100, so carrying the ramp's warmth all the way up costs a little brightness. `#fffafa` spends 1.4 L\* to land at chroma 1.76, matching `--gray90`'s 1.77 exactly, and the contrast against the page only moves from 18.3:1 to 17.7:1. A value that must be light in **both** schemes is therefore a literal `#ffffff`, never `var(--white)`; `--button-primary-color` is the worked example, and it stays a neutral white on purpose, because it labels a blue accent button in both schemes rather than sitting on the warm gray family, and the comment on it in [stylesheet/01-colors.css](stylesheet/01-colors.css) explains why the warning and success buttons do not need the same treatment. The same trap applies to any token picked for contrast: check its value in both blocks, because the name describes a role.

## The extended palette is a second, unrelated ramp

[stylesheet/01-more-colors.css](stylesheet/01-more-colors.css) defines `--color-gray-50`…`--color-gray-900` in a Spectrum-style scale, and those grays are **neutral in both schemes**. They are not the same family as `--gray00`…`--gray100` and do not follow the warm-gray rule above. Mixing the two ramps in one component is what makes a panel look subtly cold against its own page; pick one ramp per surface.
