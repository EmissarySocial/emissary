/**
 * TURBOCLICK: fires the request on MOUSEDOWN instead of waiting for the full
 * click, shaving ~100ms of perceived latency off every navigation.
 *
 * How it works:
 *
 *   1. A delegated, capture-phase MOUSEDOWN listener on document checks whether
 *      the press landed inside a .turboclick region, and if so dispatches a
 *      synthetic click() on the ELEMENT ACTUALLY PRESSED -- not on the region.
 *      A real click on the pressed element bubbles through exactly the handlers
 *      a natural click would reach, so a nested link / button / [hx-get] / and
 *      Hyperscript `on click` all fire correctly even when the .turboclick
 *      marker sits on a container that wraps several clickable children.
 *      Delegation means htmx swaps never need re-initialization.
 *
 *   2. The natural click (fired by the browser after mouseup) must then be
 *      suppressed, or every press sends TWO requests.  The discriminator is
 *      event.isTrusted: true for the browser's own click, false for our
 *      synthetic one.  The synthetic click is dispatched synchronously BEFORE
 *      the suppressor is armed, so the suppressor can only ever catch the
 *      natural click.
 *
 *   3. The suppressor is a one-shot CAPTURE-phase listener on window.  Capture
 *      at an ancestor is the only DOM mechanism guaranteed to run before
 *      htmx's own click listeners on the element itself (hx-boost / hx-get) --
 *      any bubble-phase listener on the same element only wins if it happened
 *      to register before htmx did, which is not guaranteed.
 *
 *   4. If the natural click never arrives (press, drag off the element,
 *      release), a one-shot mouseup disarms the suppressor on the next tick.
 *      The browser dispatches the natural click after mouseup handlers but
 *      before timers, so this cannot disarm too early -- it only cleans up
 *      when no click is coming, guaranteeing a stale suppressor can never
 *      swallow a later, unrelated click.
 *
 * Guards: left button only; no modifier keys (ctrl/cmd-click still opens new
 * tabs, shift-click still works); mousedowns starting inside .folder-handle
 * are ignored so SortableJS drag-to-reorder (see user-inbox/newsfeed-menu.html)
 * still starts cleanly.
 *
 * Keyboard activation (Enter/Space) fires a trusted click with no mousedown,
 * so it is never suppressed and works exactly as before.
 */

(function() {

	document.addEventListener("mousedown", function(down) {

		// RULE: Don't trigger on Right button
		if (down.button !== 0) { return; }

		// RULE: Don't trigger if any modifier keys are pressed
		if (down.ctrlKey || down.metaKey || down.shiftKey || down.altKey) { return; }

		var target = (down.target instanceof Element) ? down.target : down.target.parentElement;
		if (!target) { return; }

		// Only act on presses inside a .turboclick element
		var node = target.closest(".turboclick");
		if (!node) { return; }

		// RULE: Don't hijack drag handles.  SortableJS starts folder drags from
		// .folder-handle (see user-inbox/newsfeed-menu.html); a synthetic click
		// here would navigate instead of letting the drag begin.
		if (target.closest(".folder-handle")) { return; }

		// Fire a synthetic click on the ELEMENT ACTUALLY PRESSED (not the
		// .turboclick region).  A real click on `target` bubbles through the same
		// handlers a natural click would, so the correct nested action fires even
		// when .turboclick is on a container.  Dispatch is synchronous and
		// finishes before the suppressor below exists, so it is never suppressed.
		target.click();

		// One-time "suppressor" intercepts the natural click that follows mouseup.
		var suppressor = function(click) {

			// auto-remove ourself after first use.
			window.removeEventListener("click", suppressor, true);

			// Suppress the natural click if it lands inside the region we fired.
			if (click.isTrusted && node.contains(click.target)) {
				click.preventDefault();
				click.stopImmediatePropagation();
			}
		};

		// Attach the suppressor to the window to catch the natural click.
		window.addEventListener("click", suppressor, true);

		// Failsafe remove the "suppressor" on mouseup (in case the natural click never fires.
		window.addEventListener("mouseup", function() {
			setTimeout(function() { window.removeEventListener("click", suppressor, true); }, 0);
		}, { capture: true, once: true });

	}, true);

})();
