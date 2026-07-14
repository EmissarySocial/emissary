/*
Server Sent Events Extension
============================
This extension adds support for Server Sent Events to htmx.  See /www/extensions/sse.md for usage instructions.

*/

(function() {

	/** @type {import("../htmx").HtmxInternalApi} */
	var api;

	htmx.defineExtension("sse", {

		/**
		 * Init saves the provided reference to the internal HTMX API.
		 * 
		 * @param {import("../htmx").HtmxInternalApi} api 
		 * @returns void
		 */
		init: function(apiRef) {
			// store a reference to the internal API.
			api = apiRef;

			// set a function in the public API for creating new EventSource objects
			if (htmx.createEventSource == undefined) {
				htmx.createEventSource = createEventSource;
			}
		},

		/**
		 * onEvent handles all events passed to this extension.
		 * 
		 * @param {string} name 
		 * @param {Event} evt 
		 * @returns void
		 */
		onEvent: function(name, evt) {

			switch (name) {

				case "htmx:beforeCleanupElement":
					var internalData = api.getInternalData(evt.target)
					// Try to remove remove an EventSource when elements are removed
					if (internalData.sseEventSource) {
						internalData.sseEventSource.close();
					}

					return;

				// Try to create EventSources when elements are processed
				case "htmx:afterProcessNode":
					ensureEventSourceOnElement(evt.target);
					registerSSE(evt.target);
			}
		}
	});

	///////////////////////////////////////////////
	// HELPER FUNCTIONS
	///////////////////////////////////////////////


	/**
	 * createEventSource is the default method for creating new EventSource objects.
	 * it is hoisted into htmx.config.createEventSource to be overridden by the user, if needed.
	 * 
	 * @param {string} url 
	 * @returns EventSource
	 */
	function createEventSource(url) {
		return new EventSource(url, { withCredentials: true });
	}

	function splitOnWhitespace(trigger) {
		return trigger.trim().split(/\s+/);
	}

	function getLegacySSEURL(elt) {
		var legacySSEValue = api.getAttributeValue(elt, "hx-sse");
		if (legacySSEValue) {
			var values = splitOnWhitespace(legacySSEValue);
			for (var i = 0; i < values.length; i++) {
				var value = values[i].split(/:(.+)/);
				if (value[0] === "connect") {
					return value[1];
				}
			}
		}
	}

	function getLegacySSESwaps(elt) {
		var legacySSEValue = api.getAttributeValue(elt, "hx-sse");
		var returnArr = [];
		if (legacySSEValue != null) {
			var values = splitOnWhitespace(legacySSEValue);
			for (var i = 0; i < values.length; i++) {
				var value = values[i].split(/:(.+)/);
				if (value[0] === "swap") {
					returnArr.push(value[1]);
				}
			}
		}
		return returnArr;
	}

	/**
	 * registerSSE looks for attributes that can contain sse events, right 
	 * now hx-trigger and sse-swap and adds listeners based on these attributes too
	 * the closest event source
	 *
	 * @param {HTMLElement} elt
	 */
	function registerSSE(elt) {
		// Find closest existing event source
		var sourceElement = api.getClosestMatch(elt, hasEventSource);
		if (sourceElement == null) {
			// api.triggerErrorEvent(elt, "htmx:noSSESourceError")
			return null; // no eventsource in parentage, orphaned element
		}

		// Set internalData and source
		var internalData = api.getInternalData(sourceElement);
		var source = internalData.sseEventSource;

		// LOCAL PATCH (Emissary): registerSSE can visit the same child more than once
		// (its own htmx:afterProcessNode pass plus its sse-connect ancestor's pass, and
		// again after every EventSource reconnect).  Without cleanup, each visit stacks
		// another live listener and every SSE message fires multiple requests.  Each
		// child remembers its registrations; the first time a pass touches a child, its
		// previous registrations are removed.
		var clearedChildren = [];

		function clearOnce(child) {
			if (clearedChildren.indexOf(child) !== -1) {
				return;
			}
			clearedChildren.push(child);
			var childData = api.getInternalData(child);
			(childData.sseRegistrations || []).forEach(function(reg) {
				reg.source.removeEventListener(reg.eventName, reg.listener);
			});
			childData.sseRegistrations = [];
		}

		function remember(child, eventName, listener) {
			var childData = api.getInternalData(child);
			if (childData.sseRegistrations == null) {
				childData.sseRegistrations = [];
			}
			childData.sseRegistrations.push({ source: source, eventName: eventName, listener: listener });
		}

		// Add message handlers for every `sse-swap` attribute
		queryAttributeOnThisOrChildren(elt, "sse-swap").forEach(function(child) {

			var sseSwapAttr = api.getAttributeValue(child, "sse-swap");
			if (sseSwapAttr) {
				var sseEventNames = sseSwapAttr.split(",");
			} else {
				var sseEventNames = getLegacySSESwaps(child);
			}

			clearOnce(child); // LOCAL PATCH: drop registrations from any previous pass

			for (var i = 0; i < sseEventNames.length; i++) {
				var sseEventName = sseEventNames[i].trim();
				var listener = function(event) {

					// If the source is missing then close SSE
					if (maybeCloseSSESource(sourceElement)) {
						return;
					}

					// If the body no longer contains the element, remove the listener
					if (!api.bodyContains(child)) {
						source.removeEventListener(sseEventName, listener);
					}

					// swap the response into the DOM and trigger a notification
					swap(child, event.data);
					api.triggerEvent(elt, "htmx:sseMessage", event);
				};

				// Register the new listener
				api.getInternalData(child).sseEventListener = listener;
				remember(child, sseEventName, listener); // LOCAL PATCH: track for cleanup
				source.addEventListener(sseEventName, listener);
			}
		});

		// Add message handlers for every `hx-trigger="sse:*"` attribute.
		//
		// LOCAL PATCH (Emissary): the stock 1.9.12 extension (upstream included) declares
		// this listener but never registers it with the EventSource, so hx-trigger="sse:..."
		// silently does nothing.  This implementation completes the wiring the way the
		// htmx 2.x extension does: each EventSource message is re-published as a DOM event
		// ("sse:<name>") on the element, where htmx core's own hx-trigger listener -- and
		// any hyperscript "on sse:..." handler -- picks it up.  Unlike both stock versions,
		// the attribute is split on commas so mixed triggers work, e.g.
		//   hx-trigger="sse:message, refreshSidebar from:window"
		//   hx-trigger="load, sse:1234abcd"
		queryAttributeOnThisOrChildren(elt, "hx-trigger").forEach(function(child) {

			var triggerAttr = api.getAttributeValue(child, "hx-trigger");
			if (triggerAttr == null) {
				return;
			}

			clearOnce(child); // drop registrations from any previous pass

			triggerAttr.split(",").forEach(function(triggerSpec) {

				triggerSpec = triggerSpec.trim();

				// Only process triggers with the "sse:" prefix
				if (triggerSpec.slice(0, 4) != "sse:") {
					return;
				}

				// The SSE event name is the text after the prefix, up to any modifier
				var sseEventName = triggerSpec.substr(4).split(/\s/)[0];
				if (sseEventName === "") {
					return;
				}

				var listener = function(event) {
					if (maybeCloseSSESource(sourceElement)) {
						return;
					}

					if (!api.bodyContains(child)) {
						source.removeEventListener(sseEventName, listener);
						return;
					}

					// Re-publish as a DOM event so htmx core (hx-trigger="sse:...") and
					// hyperscript ("on sse:...") handlers can hear it.
					htmx.trigger(child, "sse:" + sseEventName, event);
					htmx.trigger(child, "htmx:sseMessage", event);
				};

				remember(child, sseEventName, listener);
				source.addEventListener(sseEventName, listener);
			});
		});
	}

	/**
	 * ensureEventSourceOnElement creates a new EventSource connection on the provided element.
	 * If a usable EventSource already exists, then it is returned.  If not, then a new EventSource
	 * is created and stored in the element's internalData.
	 * @param {HTMLElement} elt
	 * @param {number} retryCount
	 * @returns {EventSource | null}
	 */
	function ensureEventSourceOnElement(elt, retryCount) {

		if (elt == null) {
			return null;
		}

		// handle extension source creation attribute
		queryAttributeOnThisOrChildren(elt, "sse-connect").forEach(function(child) {
			var sseURL = api.getAttributeValue(child, "sse-connect");
			if (sseURL == null) {
				return;
			}

			ensureEventSource(child, sseURL, retryCount);
		});

		// handle legacy sse, remove for HTMX2
		queryAttributeOnThisOrChildren(elt, "hx-sse").forEach(function(child) {
			var sseURL = getLegacySSEURL(child);
			if (sseURL == null) {
				return;
			}

			ensureEventSource(child, sseURL, retryCount);
		});

	}

	function ensureEventSource(elt, url, retryCount) {
		var source = htmx.createEventSource(url);

		source.onerror = function(err) {

			// Log an error event
			api.triggerErrorEvent(elt, "htmx:sseError", { error: err, source: source });

			// If parent no longer exists in the document, then clean up this EventSource
			if (maybeCloseSSESource(elt)) {
				return;
			}

			// Otherwise, try to reconnect the EventSource
			if (source.readyState === EventSource.CLOSED) {
				retryCount = retryCount || 0;
				var timeout = Math.random() * (2 ^ retryCount) * 500;
				window.setTimeout(function() {
					ensureEventSourceOnElement(elt, Math.min(7, retryCount + 1));
				}, timeout);
			}
		};

		source.onopen = function(evt) {
			api.triggerEvent(elt, "htmx:sseOpen", { source: source });
		}

		api.getInternalData(elt).sseEventSource = source;

		// LOCAL PATCH (Emissary): re-attach listeners whenever a source is (re)created.
		// The stock snapshot never re-registered after a reconnect (source.onerror makes
		// a NEW EventSource), so SSE went permanently deaf after any server restart or
		// proxy idle-timeout.  Registration is idempotent (see clearOnce/remember above).
		registerSSE(elt);
	}

	/**
	 * maybeCloseSSESource confirms that the parent element still exists.
	 * If not, then any associated SSE source is closed and the function returns true.
	 * 
	 * @param {HTMLElement} elt 
	 * @returns boolean
	 */
	function maybeCloseSSESource(elt) {
		if (!api.bodyContains(elt)) {
			var source = api.getInternalData(elt).sseEventSource;
			if (source != undefined) {
				source.close();
				// source = null
				return true;
			}
		}
		return false;
	}

	/**
	 * queryAttributeOnThisOrChildren returns all nodes that contain the requested attributeName, INCLUDING THE PROVIDED ROOT ELEMENT.
	 * 
	 * @param {HTMLElement} elt 
	 * @param {string} attributeName 
	 */
	function queryAttributeOnThisOrChildren(elt, attributeName) {

		var result = [];

		// If the parent element also contains the requested attribute, then add it to the results too.
		if (api.hasAttribute(elt, attributeName)) {
			result.push(elt);
		}

		// Search all child nodes that match the requested attribute
		elt.querySelectorAll("[" + attributeName + "], [data-" + attributeName + "]").forEach(function(node) {
			result.push(node);
		});

		return result;
	}

	/**
	 * @param {HTMLElement} elt
	 * @param {string} content 
	 */
	function swap(elt, content) {

		api.withExtensions(elt, function(extension) {
			content = extension.transformResponse(content, null, elt);
		});

		var swapSpec = api.getSwapSpecification(elt);
		var target = api.getTarget(elt);
		var settleInfo = api.makeSettleInfo(elt);

		api.selectAndSwap(swapSpec.swapStyle, target, elt, content, settleInfo);

		settleInfo.elts.forEach(function(elt) {
			if (elt.classList) {
				elt.classList.add(htmx.config.settlingClass);
			}
			api.triggerEvent(elt, 'htmx:beforeSettle');
		});

		// Handle settle tasks (with delay if requested)
		if (swapSpec.settleDelay > 0) {
			setTimeout(doSettle(settleInfo), swapSpec.settleDelay);
		} else {
			doSettle(settleInfo)();
		}
	}

	/**
	 * doSettle mirrors much of the functionality in htmx that 
	 * settles elements after their content has been swapped.
	 * TODO: this should be published by htmx, and not duplicated here
	 * @param {import("../htmx").HtmxSettleInfo} settleInfo 
	 * @returns () => void
	 */
	function doSettle(settleInfo) {

		return function() {
			settleInfo.tasks.forEach(function(task) {
				task.call();
			});

			settleInfo.elts.forEach(function(elt) {
				if (elt.classList) {
					elt.classList.remove(htmx.config.settlingClass);
				}
				api.triggerEvent(elt, 'htmx:afterSettle');
			});
		}
	}

	function hasEventSource(node) {
		return api.getInternalData(node).sseEventSource != null;
	}

})();
