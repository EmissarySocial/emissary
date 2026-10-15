# Scrollspy

This widget (widgetId `scrollspy`) adds an intra-page navigation menu to a stream: it scans the page's main content for headings, builds a linked outline of them, and highlights the entry for the section currently in view as the reader scrolls. It is entirely client-side, so it works on any stream whose content renders headings.

[widget.hjson](widget.hjson) declares no schema, form, or bundle. [widget.html](widget.html) is inline CSS for the sticky `#scrollspy` container plus the empty div itself, which carries `script="install Scrollspy"`. The `Scrollspy` behavior lives in `theme-global`'s [hyperscript/scrollspy._hs](../theme-global/hyperscript/scrollspy._hs), which every page loads. On `init` it finds `h1`–`h4` elements inside the first `.e-content` node, gives each a random id when it lacks one, and appends `.scrollspy-item` anchors indented by heading level (`scrollspy_indent`). It then re-runs the position check (`scrollspy_select`) on window scroll (debounced 20ms) and on clicks in the outline, using `measure` to find the first heading in the viewport and `take .selected` to move the highlight. To support more heading levels or a different content container, edit the selectors in the `_hs` file and the `.level-*` styles in widget.html.

## The behavior is installed on the element, never loaded by a script src

Hyperscript fetches `<script type="text/hyperscript" src=…>` only once, when the document first loads. After an htmx swap it reads a new script tag's inner text, which for a `src` tag is empty, so a widget that loaded its code that way worked only on a page opened directly, and showed an empty outline after any click-through. Installing a behavior from the always-loaded `theme-global` bundle runs `init` on every swap, like every other behavior in Emissary.

htmx does not clean up hyperscript listeners when it swaps `<main>`, so the outline's `on scroll from window` listener would outlive it. The handler checks `my isConnected` and calls `_hyperscript.cleanup(me)` on the first scroll after its outline is gone. Keep that guard on any behavior that listens to `window` or `document`.

## Each outline link is built as an element, never as a string

`put` in hyperscript inserts a string with `insertAdjacentHTML`, which parses it as markup, but inserts an Element with `insertAdjacentElement`, which does not. A heading's `textContent` is its *decoded* text, so a heading that reads `&lt;img src=x onerror=…&gt;` in the page has the textContent `<img src=x onerror=…>`. Pasting that into a string of HTML turned escaped text back into live markup, past the sanitizer (BUG-277). [scrollspy._hs](../theme-global/hyperscript/scrollspy._hs) therefore creates each `<a>` with `document.createElement` and assigns `className`, `href`, and `textContent` as properties. Keep it that way, and apply the same rule to any other hyperscript that builds markup from page text.
