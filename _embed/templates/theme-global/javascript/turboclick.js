/**
 * TURBOCLICK: fires the request on MOUSEDOWN instead of waiting for the full
 * click, shaving ~100ms of perceived latency off every navigation.
 *
 * How it works:
 *
 *   1. A delegated, capture-phase MOUSEDOWN listener on document finds the
 *      nearest .turboclick ancestor and dispatches a synthetic click() on it
 *      immediately.  Delegation means htmx swaps never need re-initialization.
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
 * are ignored so SortableJS drag-to-reorder (see user-inbox/menu.html) still
 * starts cleanly.
 *
 * Keyboard activation (Enter/Space) fires a trusted click with no mousedown,
 * so it is never suppressed and works exactly as before.
 */

(function() {

	document.addEventListener("mousedown", function(down) {

		// Left button only, no modifier keys (preserve open-in-new-tab, etc.)
		if (down.button !== 0) { return; }
		if (down.ctrlKey || down.metaKey || down.shiftKey || down.altKey) { return; }

		var target = (down.target instanceof Element) ? down.target : down.target.parentElement;
		if (!target) { return; }

		// Only act on presses inside a .turboclick element
		var node = target.closest(".turboclick");
		if (!node) { return; }

		// Don't hijack drag handles (SortableJS starts folder drags from .folder-handle)
		if (target.closest(".folder-handle")) { return; }

		// 1) Fire the synthetic click NOW.  Dispatch is synchronous and finishes
		//    before the suppressor below exists, so it is never suppressed itself.
		node.click();

		// 2) Arm a one-shot suppressor for the natural click that follows mouseup.
		var suppress = function(click) {
			window.removeEventListener("click", suppress, true);
			if (click.isTrusted && node.contains(click.target)) {
				click.preventDefault();
				click.stopImmediatePropagation();
			}
		};
		window.addEventListener("click", suppress, true);

		// 3) Disarm once the gesture is over, in case the natural click never fires.
		window.addEventListener("mouseup", function() {
			setTimeout(function() { window.removeEventListener("click", suppress, true); }, 0);
		}, { capture: true, once: true });

	}, true);

})();
