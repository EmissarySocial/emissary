/**
 * This file is the browser side of the multi-file zone that the edit-attachments step draws.
 *
 * The form library's html-remote draws only an empty shell that loads from the host.  The step
 * draws the zone and everything inside it: the tile row, the plus after it, the hidden single-file
 * input, and the status line.  On a saved object the shell loads from the step's own action, whose
 * row carries data-url and every saved file as a tile, and which answers every change with the row
 * again; on a new object it loads from the same step through the parent's add, whose row carries
 * data-staging-url and posts each file there, and which answers with that file's tile.  The plus
 * sits outside the row, so a swap of the row never touches it, and the zone is swapped only by
 * the reload after a cancel, which drops the state a cancel discards.  This script adds what the steps cannot:
 * taking one file at a time from the plus's picker or a drop, uploading it, drawing the tile that
 * is not saved yet with a bar across it while its file is sent, hiding the plus once the field is
 * full, and sorting by drag or keyboard.  A row marked data-readonly is left alone.
 *
 * A tile is its own handle.  A press anywhere on it but its X drags it, and with focus the arrow
 * keys move it.  The X comes with the mouse, through the theme's hover-trigger and hover-show
 * classes, and with keyboard focus inside the tile, so focus always moves to a tile, never to an X.
 *
 * Files are added one at a time.  The picker takes one, and a drop of several takes the first
 * and names the rest in the status line.  Each file is its own request, and one that arrives
 * during another waits in the script's queue with no tile until its turn.
 *
 * The row's URL tells the two modes apart, and every file uploads at once in both.  With
 * data-url the object is saved: an added file uploads through the row, whose response replaces
 * the row; the X on a saved tile is htmx's own request; and a reorder of the saved files posts
 * the order.  With data-staging-url the object is not saved yet: an added file uploads to the
 * staging route, whose response is the tile alone, appended to the row; the X on a staged tile
 * posts its delete there, and the empty answer removes the tile; and a reorder posts nothing,
 * because each tile's hidden order input moves with it and the form's Save carries them all.
 *
 * The limits, the accepted types, and the field name all come from the row, in both modes.  The
 * widget's own changes go through outside the limits, so a short field can grow and a full one
 * can be sorted, but the form's Save is stopped until the files are back within them.
 *
 * The widget's requests are its own, not the form's.  They go through htmx.ajax with the row as
 * their source, so they take the row's hx-params, hx-encoding, and hx-sync, which queues them on
 * the zone and never with the form's Save; the Save never waits for them.  While the form's Save
 * is in flight, the zone takes no new file and starts no queued upload.
 */

(function() {

	if (window.uploadMultiple) {
		return
	}

	const SORTABLE_URL = "/.themes/global/resources/sortable-1.15.7/Sortable.min.js"

	/** SAVING_MESSAGE is what the status line says when a file arrives during the form's Save */
	const SAVING_MESSAGE = "Wait for the Save to finish before adding a file"

	/** states holds each zone's working state, keyed by the zone element */
	const states = new WeakMap()

	/** waitingForSortable holds the row initializers to run once Sortable has loaded */
	let waitingForSortable = []

	/** stateOf returns the zone's state, making it on the first call */
	function stateOf(zone) {

		let state = states.get(zone)

		if (state === undefined) {
			state = {
				busy: 0,            // Requests from this zone that are in flight
				fired: [],          // The kinds of the requests this script has fired, in order, until htmx starts each one
				active: null,       // The kind of the request in flight: "upload", "sort", "reload", "delete", or "other"
				inflight: null,     // The upload in flight: its file and its pending tile
				queue: [],          // Files waiting for the upload in flight to finish
				cancelling: false,  // Whether the upload in flight was cancelled by its X
				savedOrder: [],     // The saved files' IDs in the order the row was last drawn
				refocus: null,      // The ID of the tile that takes focus after the next swap
				deleteFocus: null,  // The ID of the tile that takes focus after a delete, or null for the plus
				depth: 0,           // Drag enters minus drag leaves, so a child's dragleave does not clear the highlight
				saving: false,      // Whether the form around the zone has a Save in flight
			}
			states.set(zone, state)
		}

		return state
	}

	/** zoneOf returns the widget that holds the element */
	function zoneOf(element) {
		return element.closest("[data-upload-multiple]")
	}

	/** rowOf returns the zone's tile row */
	function rowOf(zone) {
		return zone.querySelector(".upload-multiple-tiles")
	}

	/** inputOf returns the zone's hidden file input */
	function inputOf(zone) {
		return zone.querySelector("input[type=file]")
	}

	/** addOf returns the zone's plus tile, which opens the picker */
	function addOf(zone) {
		return zone.querySelector(".upload-multiple-add")
	}

	/** isLive reports whether the zone belongs to a saved object, whose own route takes every change */
	function isLive(zone) {

		const row = rowOf(zone)

		return row !== null && row.hasAttribute("data-url")
	}

	/** requestURL returns the route the zone's uploads go to: the object's own, or the staging route of a new one */
	function requestURL(zone) {

		const row = rowOf(zone)

		return isLive(zone) ? row.dataset.url : row.dataset.stagingUrl
	}

	/** isEditable reports whether the zone takes changes, which a read-only row never does */
	function isEditable(zone) {

		const row = rowOf(zone)

		return row !== null && row.dataset.readonly !== "true"
	}

	/** announce writes a message to the zone's status line */
	function announce(zone, message, warning) {

		const status = zone.querySelector(".upload-multiple-status")

		if (status === null) {
			return
		}

		status.textContent = message
		status.classList.toggle("warning", warning === true)
	}

	/** tilesOf returns the zone's tiles in DOM order */
	function tilesOf(zone) {
		return Array.from(zone.querySelectorAll(".upload-multiple-tile"))
	}

	/** savedOrder returns the IDs of the saved tiles in DOM order */
	function savedOrder(zone) {
		return tilesOf(zone).filter((tile) => tile.dataset.value).map((tile) => tile.dataset.value)
	}

	/** limitsOf returns the zone's limits, which the step's row carries; before the row arrives there are none */
	function limitsOf(zone) {

		const row = rowOf(zone)
		const source = row !== null ? row.dataset : {}
		const minimum = parseInt(source.minimum, 10)
		const maximum = parseInt(source.maximum, 10)

		return {
			minimum: isNaN(minimum) ? 0 : minimum,
			maximum: isNaN(maximum) ? Infinity : maximum,
		}
	}

	/** acceptsFile reports whether the zone's accept list admits the file's type or extension */
	function acceptsFile(zone, file) {

		const row = rowOf(zone)
		const accept = (row !== null && row.dataset.accept) || ""

		if (accept.trim() === "") {
			return true
		}

		const type = (file.type || "").toLowerCase()
		const name = (file.name || "").toLowerCase()

		return accept.split(",").some(function(pattern) {

			pattern = pattern.trim().toLowerCase()

			if (pattern === "") {
				return false
			}

			if (pattern.startsWith(".")) {
				return name.endsWith(pattern)
			}

			if (pattern.endsWith("/*")) {
				return type.startsWith(pattern.slice(0, -1))
			}

			return type === pattern
		})
	}

	/** sameFile reports whether two files are the same pick: the same name, size, and modified time */
	function sameFile(a, b) {
		return a.name === b.name && a.size === b.size && a.lastModified === b.lastModified
	}

	/** heldFiles returns the files the zone holds but has not uploaded: the upload in flight and the queue */
	function heldFiles(zone) {

		const state = stateOf(zone)
		const result = []

		if (state.inflight !== null) {
			result.push(state.inflight.file)
		}

		result.push(...state.queue)

		return result
	}

	/** plural returns the count with the noun, in the singular or the plural */
	function plural(count, noun) {
		return count + " " + noun + (count === 1 ? "" : "s")
	}

	/** takeFile picks the one file the zone takes from a drop or the picker, and announces what it left and why */
	function takeFile(zone, files) {

		const held = heldFiles(zone)
		const room = limitsOf(zone).maximum - tilesOf(zone).length - stateOf(zone).queue.length
		const skipped = []
		const extras = []
		let taken = null

		// RULE: One file per add. The first file that fits is taken, and the rest are named.
		Array.from(files).forEach(function(file) {

			if (taken !== null) {
				extras.push(file.name)
				return
			}

			if (!acceptsFile(zone, file)) {
				skipped.push(file.name + " (not an accepted type)")
				return
			}

			if (held.some((other) => sameFile(file, other))) {
				skipped.push(file.name + " (already here)")
				return
			}

			if (room < 1) {
				skipped.push(file.name + " (does not fit)")
				return
			}

			taken = file
		})

		if (extras.length > 0) {
			skipped.push(extras.join(", ") + " (one file at a time)")
		}

		if (skipped.length > 0) {
			announce(zone, "Skipped " + skipped.join(", "), true)
		} else if (taken !== null) {
			announce(zone, "")
		}

		return taken
	}

	/** previewURL returns an object URL for an image, or null for anything else */
	function previewURL(file) {

		if ((file.type || "").startsWith("image/")) {
			return URL.createObjectURL(file)
		}

		return null
	}

	/** revokePreview releases the tile's object URL, if it has one */
	function revokePreview(tile) {

		const preview = tile.querySelector(".upload-multiple-preview")

		if (preview !== null && preview.src.startsWith("blob:")) {
			URL.revokeObjectURL(preview.src)
		}
	}

	/** drawPendingTile makes the tile of a file being sent: a preview, an X that cancels, and a bar */
	function drawPendingTile(file) {

		const tile = document.createElement("div")
		tile.className = "upload-multiple-tile upload-multiple-pending hover-trigger"

		const url = previewURL(file)

		if (url !== null) {
			const preview = document.createElement("img")
			preview.className = "upload-multiple-preview"
			preview.src = url
			preview.alt = file.name
			preview.draggable = false
			tile.appendChild(preview)
		} else {
			const icon = document.createElement("i")
			icon.className = "bi bi-file-earmark upload-multiple-icon"
			tile.appendChild(icon)

			const name = document.createElement("span")
			name.className = "upload-multiple-filename"
			name.textContent = file.name
			tile.appendChild(name)
		}

		const remove = document.createElement("button")
		remove.type = "button"
		remove.className = "upload-multiple-remove hover-show"
		remove.setAttribute("aria-label", "Cancel " + file.name)
		remove.innerHTML = '<i class="bi bi-x-lg"></i>'
		tile.appendChild(remove)

		const progress = document.createElement("div")
		progress.className = "upload-multiple-progress"
		progress.setAttribute("role", "progressbar")
		progress.setAttribute("aria-label", "Uploading " + file.name)
		progress.setAttribute("aria-valuemin", "0")
		progress.setAttribute("aria-valuemax", "100")
		progress.setAttribute("aria-valuenow", "0")
		progress.appendChild(document.createElement("div"))
		tile.appendChild(progress)

		return tile
	}

	/** setBar fills a tile's bar to the percent, for the eye and for the screen reader */
	function setBar(tile, percent) {

		const progress = tile.querySelector(".upload-multiple-progress")

		if (progress === null) {
			return
		}

		progress.firstElementChild.style.width = percent + "%"
		progress.setAttribute("aria-valuenow", String(percent))
	}

	/** intake takes one file from a drop or the picker and queues it for upload */
	function intake(zone, files) {

		const state = stateOf(zone)

		if (!isEditable(zone)) {
			return
		}

		// RULE: A file added during the form's Save would miss it, so the zone takes none until the Save settles
		if (state.saving) {
			announce(zone, SAVING_MESSAGE, true)
			return
		}

		const file = takeFile(zone, files)

		if (file === null) {
			return
		}

		state.queue.push(file)
		uploadNext(zone)
	}

	/** fire sends one of the zone's own requests through the row, remembering its kind for the events that follow */
	function fire(zone, kind, method, options) {

		const state = stateOf(zone)
		const row = rowOf(zone)

		// A saved object answers a change with the row, and a new object's staging route with one tile, which
		// joins the row; a reload asks the object's route for the whole zone again, which replaces this one
		const reload = kind === "reload"
		const swap = reload ? "outerHTML" : (isLive(zone) ? "outerHTML" : "beforeend")

		state.fired.push(kind)
		holdZone(zone)
		htmx.ajax(method, requestURL(zone), Object.assign({source: row, target: reload ? zone : row, swap: swap}, options))
	}

	/** uploadNext posts the next waiting file through the row, with a pending tile, unless one is already in flight */
	function uploadNext(zone) {

		const state = stateOf(zone)

		if (state.inflight !== null || state.queue.length === 0) {
			return
		}

		// RULE: A queued file waits while the form's Save is in flight, and starts once it settles
		if (state.saving) {
			return
		}

		const file = state.queue.shift()
		const tile = drawPendingTile(file)

		rowOf(zone).appendChild(tile)
		applyLimits(zone)

		state.inflight = {
			file: file,
			tile: tile,
			done: false,
		}

		// The file travels as a request value, never through the input, so a queued sort cannot pick it up
		const values = {}
		values[rowOf(zone).dataset.name] = [file]

		fire(zone, "upload", "POST", {values: values})
	}

	/** finishUpload clears the upload in flight and drops its pending tile, which a replaced row has already taken away */
	function finishUpload(zone, message) {

		const state = stateOf(zone)
		const inflight = state.inflight

		if (inflight === null) {
			return
		}

		state.inflight = null
		state.cancelling = false
		revokePreview(inflight.tile)
		inflight.tile.remove()
		applyLimits(zone)

		if (message !== "") {
			announce(zone, message, true)
		}

		uploadNext(zone)
	}

	/** applyLimits mirrors the limits: the plus goes at the maximum, the saved Xs lock at the minimum, and the status line counts what is missing or extra */
	function applyLimits(zone) {

		const limits = limitsOf(zone)
		const count = tilesOf(zone).length
		const add = addOf(zone)
		const full = count >= limits.maximum

		// The plus goes at the maximum, and stays in place but disabled while the form's Save is in flight
		if (add !== null) {
			add.disabled = full || stateOf(zone).saving
			add.hidden = full
		}

		// At the minimum, deleting a saved file would be refused, so the X says so before the request.
		// A staged file's X never locks, because a new object's minimum is its Save's to enforce.
		const atMinimum = isLive(zone) && limits.minimum > 0 && count <= limits.minimum

		tilesOf(zone).forEach(function(tile) {

			if (!tile.dataset.value) {
				return
			}

			const remove = tile.querySelector(".upload-multiple-remove")

			if (remove === null) {
				return
			}

			remove.title = atMinimum ? "Add another file before removing this one" : ""

			// The minimum owns a disabled X outright; otherwise an X held for a request in flight stays held
			if (atMinimum) {
				remove.disabled = true
				delete remove.dataset.uploadMultipleHeld
			} else if (remove.dataset.uploadMultipleHeld !== "true") {
				remove.disabled = false
			}
		})

		// A read-only row has no limits to report
		if (!isEditable(zone)) {
			return
		}

		if (count < limits.minimum) {
			announce(zone, plural(limits.minimum - count, "more file") + " needed")
		} else if (count > limits.maximum) {
			announce(zone, plural(count - limits.maximum, "file") + " over the limit")
		}
	}

	/** limitMessage returns why the zone's files cannot be saved, in the step's own words, or "" when their count is within its limits */
	function limitMessage(zone) {

		// A read-only row has no limits to keep
		if (!isEditable(zone)) {
			return ""
		}

		const limits = limitsOf(zone)
		const count = savedOrder(zone).length

		if (count < limits.minimum) {
			return "At least " + plural(limits.minimum, "file") + (limits.minimum === 1 ? " is" : " are") + " required"
		}

		if (count > limits.maximum) {
			return "At most " + plural(limits.maximum, "file") + (limits.maximum === 1 ? " is" : " are") + " allowed"
		}

		return ""
	}

	/** guardSubmit stops a form's Save while a widget inside it holds too few or too many files, and says why beside the Save */
	function guardSubmit(event) {

		const form = event.target instanceof HTMLFormElement ? event.target : null

		if (form === null) {
			return
		}

		const zone = Array.from(form.querySelectorAll("[data-upload-multiple]")).find((other) => limitMessage(other) !== "")

		if (zone === undefined) {
			return
		}

		// RULE: Stopped in the capture phase, the submit never reaches htmx's listener on the form, so nothing is sent
		event.preventDefault()
		event.stopImmediatePropagation()

		// Say why in the zone, and where the server's own refusal would land, since the zone may sit in a hidden tab
		const message = limitMessage(zone)
		announce(zone, message, true)

		const span = form.querySelector("#htmx-response-message")

		if (span !== null) {
			const text = document.createElement("span")
			text.className = "text-red"
			text.textContent = message
			span.replaceChildren(text)
		}
	}

	/** zonesSavedBy returns the widgets inside the form whose own request raised the event, or none for any other request */
	function zonesSavedBy(event) {

		// RULE: Only the form's own request names the form; a request made inside it names its own element
		const form = event.detail && event.detail.elt

		if (!(form instanceof HTMLFormElement)) {
			return []
		}

		return Array.from(form.querySelectorAll("[data-upload-multiple]"))
	}

	/** onFormSending stops every widget inside a form from taking files while the form's Save is in flight */
	function onFormSending(event) {

		zonesSavedBy(event).forEach(function(zone) {
			stateOf(zone).saving = true
			applyLimits(zone)
		})
	}

	/** onFormSettled lets the form's widgets take files again once its Save settles, and starts any upload that waited */
	function onFormSettled(event) {

		zonesSavedBy(event).forEach(function(zone) {

			const status = zone.querySelector(".upload-multiple-status")

			stateOf(zone).saving = false

			if (status !== null && status.textContent === SAVING_MESSAGE) {
				announce(zone, "")
			}

			applyLimits(zone)
			uploadNext(zone)
		})
	}

	/** zoneBusy reports whether the zone has a request in flight or one fired and not yet started */
	function zoneBusy(zone) {

		const state = stateOf(zone)

		return state.busy > 0 || state.fired.length > 0
	}

	/** holdButton disables a button for the length of a request, marked so that releaseButton knows it was this script */
	function holdButton(button) {

		if (!button.disabled) {
			button.disabled = true
			button.dataset.uploadMultipleHeld = "true"
		}
	}

	/** releaseButton enables a button that holdButton disabled, and leaves one disabled for another reason alone */
	function releaseButton(button) {

		if (button.dataset.uploadMultipleHeld === "true") {
			button.disabled = false
			delete button.dataset.uploadMultipleHeld
		}
	}

	/** removeButtonsOf returns the X of every saved or staged tile in the zone */
	function removeButtonsOf(zone) {

		return tilesOf(zone)
			.filter((tile) => tile.dataset.value)
			.map((tile) => tile.querySelector(".upload-multiple-remove"))
			.filter((button) => button !== null)
	}

	/** holdZone marks the zone busy and keeps its tiles from deleting while its request runs; the form around it is never held */
	function holdZone(zone) {

		// RULE: The Xs are held because htmx would queue a delete behind the request on the zone, and on a saved
		// object the response replaces the row, so the queued delete would belong to a button that is gone and be dropped
		removeButtonsOf(zone).forEach(holdButton)
		zone.dataset.uploadMultipleBusy = "true"
	}

	/** releaseZone lets the zone's tiles delete again once it is idle */
	function releaseZone(zone) {

		if (zoneBusy(zone)) {
			return
		}

		removeButtonsOf(zone).forEach(releaseButton)
		delete zone.dataset.uploadMultipleBusy
	}

	/** requestOfZone reports whether an htmx event belongs to the zone: its row, a button in the row, or the zone itself once the row was swapped out */
	function requestOfZone(zone, event) {

		const element = event.detail && event.detail.elt

		if (!(element instanceof Element)) {
			return false
		}

		return element === zone || zone.contains(element)
	}

	/** neighborOf returns the ID of the tile after this one, or before it, or null when it is alone */
	function neighborOf(zone, tile) {

		const tiles = tilesOf(zone)
		const index = tiles.indexOf(tile)
		const neighbor = tiles[index + 1] || tiles[index - 1] || null

		return neighbor !== null && neighbor.dataset.value ? neighbor.dataset.value : null
	}

	/** onBeforeRequest marks the zone busy, names the request's kind, and dims the tile whose X is deleting it */
	function onBeforeRequest(zone, event) {

		if (!requestOfZone(zone, event)) {
			return
		}

		const state = stateOf(zone)
		const tile = event.detail.elt.closest(".upload-multiple-tile")

		state.busy++

		// A request from the row is one this script fired, in order; one from a tile is its X
		if (tile !== null) {
			tile.classList.add("upload-multiple-busy")
			state.active = "delete"
			state.deleteFocus = neighborOf(zone, tile)
		} else {
			state.active = state.fired.shift() || "other"
		}

		holdZone(zone)
	}

	/** onAfterRequest settles the request: a response that reached the row or a tile speaks for itself, and a failure is said in the status line */
	function onAfterRequest(zone, event) {

		if (!requestOfZone(zone, event)) {
			return
		}

		const state = stateOf(zone)
		const detail = event.detail
		const kind = state.active
		const target = detail.target instanceof Element ? detail.target : null

		// A validation error is answered 200 and retargeted to the form's message span, so the target tells success from refusal
		const ok = detail.successful === true && target !== null && (target.classList.contains("upload-multiple-tiles") || target.classList.contains("upload-multiple-tile") || target.hasAttribute("data-upload-multiple"))

		state.busy = Math.max(state.busy - 1, 0)
		state.active = null

		const message = (detail.successful === true && !ok) ? inlineMessage() : ""

		if (kind === "upload") {
			finishUpload(zone, (ok || state.cancelling) ? "" : (message || "The file was not uploaded"))
		} else if (ok) {
			announce(zone, "")
		} else if (kind === "delete") {
			tilesOf(zone).forEach((tile) => tile.classList.remove("upload-multiple-busy"))
			announce(zone, message || "The file was not deleted", true)
		} else if (kind === "sort") {
			announce(zone, message || "The new order was not saved", true)
		} else if (message !== "") {
			announce(zone, message, true)
		}

		releaseZone(zone)

		// Focus moves only once the delete has settled and the zone is released
		if (ok && kind === "delete") {
			focusAfterDelete(zone)
		}
	}

	/** focusAfterDelete moves focus to the deleted tile's neighbor, or to the plus when none is left */
	function focusAfterDelete(zone) {

		const state = stateOf(zone)
		const neighbor = state.deleteFocus !== null ? tilesOf(zone).find((tile) => tile.dataset.value === state.deleteFocus) : undefined
		const target = neighbor !== undefined ? neighbor : addOf(zone)

		state.deleteFocus = null

		if (target !== null) {
			target.focus()
		}
	}

	/** inlineMessage returns the text htmx put in the form's inline message span, which is where a validation error lands */
	function inlineMessage() {

		const span = document.getElementById("htmx-response-message")

		return span === null ? "" : span.textContent.trim()
	}

	/** onProgress moves the pending tile's bar as the upload's bytes go */
	function onProgress(zone, event) {

		if (!requestOfZone(zone, event)) {
			return
		}

		const state = stateOf(zone)
		const inflight = state.inflight
		const detail = event.detail

		if (state.active !== "upload" || inflight === null || inflight.done || detail.lengthComputable !== true || detail.total === 0) {
			return
		}

		// htmx fires this for the response download too, so stop once the upload has reached its total
		if (detail.loaded >= detail.total) {
			inflight.done = true
		}

		setBar(inflight.tile, Math.round(Math.min(detail.loaded / detail.total, 1) * 100))
	}

	/** cancelUpload aborts the upload in flight and drops the queue; a saved object's row is reloaded, so the screen matches whatever the server kept */
	function cancelUpload(zone) {

		const state = stateOf(zone)

		if (state.inflight === null) {
			return
		}

		// RULE: The abort goes to the zone, because hx-sync keeps the request on the sync element, and htmx
		// aborts only the request kept on the element the event reaches
		state.queue = []
		state.cancelling = true
		htmx.trigger(zone, "htmx:abort")

		if (isLive(zone)) {
			fire(zone, "reload", "GET", {})
		}
	}

	/** moveTile moves a tile to the given index among its siblings, and announces its new place */
	function moveTile(zone, tile, index) {

		const row = rowOf(zone)
		const tiles = tilesOf(zone).filter((other) => other !== tile)
		const bounded = Math.min(Math.max(index, 0), tiles.length)

		if (bounded >= tiles.length) {
			row.appendChild(tile)
		} else {
			row.insertBefore(tile, tiles[bounded])
		}

		announcePlace(zone, tile)
		afterReorder(zone, tile)
	}

	/** tileName returns the file's name from its tile: the preview's alt, or the filename span */
	function tileName(tile) {

		const preview = tile.querySelector(".upload-multiple-preview")

		if (preview !== null && preview.alt) {
			return preview.alt
		}

		const filename = tile.querySelector(".upload-multiple-filename")

		return filename !== null ? filename.textContent : "File"
	}

	/** announcePlace reads out where the tile now sits, as "photo.jpg, 2 of 5" */
	function announcePlace(zone, tile) {

		const tiles = tilesOf(zone)

		announce(zone, tileName(tile) + ", " + (tiles.indexOf(tile) + 1) + " of " + tiles.length)
	}

	/** afterReorder posts the saved files' new order on a saved object; on a new one the order inputs moved with the tiles, and the Save carries them */
	function afterReorder(zone, tile) {

		const state = stateOf(zone)

		if (!isLive(zone)) {
			return
		}

		const order = savedOrder(zone)

		if (order.join(",") === state.savedOrder.join(",")) {
			return
		}

		state.savedOrder = order
		state.refocus = tile !== null && tile.dataset.value ? tile.dataset.value : null
		fire(zone, "sort", "POST", {})
	}

	/** onKeydown moves the focused tile with the arrow keys, and to either end with Home and End */
	function onKeydown(zone, event) {

		// The tile itself is the handle, so a key pressed on its X or anything else inside it is left alone
		const tile = event.target instanceof Element && event.target.matches(".upload-multiple-tile") ? event.target : null

		if (tile === null) {
			return
		}

		const tiles = tilesOf(zone)
		const index = tiles.indexOf(tile)
		let target = null

		switch (event.key) {
			case "ArrowLeft":
			case "ArrowUp":
				target = index - 1
				break
			case "ArrowRight":
			case "ArrowDown":
				target = index + 1
				break
			case "Home":
				target = 0
				break
			case "End":
				target = tiles.length - 1
				break
			default:
				return
		}

		event.preventDefault()

		if (target < 0 || target >= tiles.length || target === index) {
			return
		}

		moveTile(zone, tile, target)
		tile.focus()
	}

	/** hasFiles reports whether a drag carries files from outside the page, which a Sortable drag never does */
	function hasFiles(event) {
		return event.dataTransfer !== null && event.dataTransfer !== undefined && Array.from(event.dataTransfer.types).includes("Files")
	}

	/** onClick runs the zone's own buttons: the plus, and a pending tile's cancel; every other X is htmx's own request */
	function onClick(zone, event) {

		const button = event.target instanceof Element ? event.target.closest("button") : null

		if (button === null || !zone.contains(button)) {
			return
		}

		if (button.classList.contains("upload-multiple-add")) {
			const input = inputOf(zone)
			if (input !== null) {
				input.click()
			}
			return
		}

		if (!button.classList.contains("upload-multiple-remove")) {
			return
		}

		const tile = button.closest(".upload-multiple-tile")

		if (tile !== null && tile.classList.contains("upload-multiple-pending")) {
			cancelUpload(zone)
		}
	}

	/** initZone wires a widget once: its drag and drop, its picker, its buttons, its keyboard, and the htmx events from its row */
	function initZone(zone) {

		if (zone.dataset.uploadMultipleReady === "true") {
			return
		}

		zone.dataset.uploadMultipleReady = "true"

		const state = stateOf(zone)

		zone.addEventListener("dragenter", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			state.depth++
			zone.classList.toggle("highlight", !state.saving)
		})

		// RULE: While the form's Save is in flight, the drop is refused before it lands
		zone.addEventListener("dragover", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			if (state.saving) {
				event.dataTransfer.dropEffect = "none"
			}
		})

		zone.addEventListener("dragleave", function(event) {
			if (!hasFiles(event)) {
				return
			}
			state.depth = Math.max(state.depth - 1, 0)
			if (state.depth === 0) {
				zone.classList.remove("highlight")
			}
		})

		zone.addEventListener("drop", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			state.depth = 0
			zone.classList.remove("highlight")
			intake(zone, event.dataTransfer.files)
		})

		// The picker's file is taken out of the input at once, so the form's Save never carries it. The
		// input arrives with the zone's contents, after this runs, so the listener sits on the zone.
		zone.addEventListener("change", function(event) {
			const input = inputOf(zone)
			if (input === null || event.target !== input) {
				return
			}
			const files = Array.from(input.files)
			input.value = ""
			intake(zone, files)
		})

		zone.addEventListener("click", (event) => onClick(zone, event))
		zone.addEventListener("keydown", (event) => onKeydown(zone, event))
		zone.addEventListener("htmx:beforeRequest", (event) => onBeforeRequest(zone, event))
		zone.addEventListener("htmx:afterRequest", (event) => onAfterRequest(zone, event))
		zone.addEventListener("htmx:xhr:progress", (event) => onProgress(zone, event))

		applyLimits(zone)
	}

	/** initRow readies a tile row after it is drawn or swapped in: Sortable, the limits, and the tile that had focus */
	function initRow(row) {

		const zone = zoneOf(row)

		// A read-only row has no controls and must not sort, so nothing is wired to it
		if (zone === null || row.dataset.readonly === "true") {
			return
		}

		initZone(zone)

		const state = stateOf(zone)

		state.savedOrder = savedOrder(zone)
		applyLimits(zone)

		withSortable(function() {

			if (!row.isConnected || row.uploadMultipleSortable) {
				return
			}

			// The tile is its own handle: a press anywhere on it drags, except on its X, which keeps
			// its press, and on a pending tile, which is not placed until its file has landed
			row.uploadMultipleSortable = new Sortable(row, {
				draggable: ".upload-multiple-tile",
				filter: ".upload-multiple-pending, .upload-multiple-remove",
				preventOnFilter: false,
				animation: 150,
				ghostClass: "draggable-whisper",
				delay: 150,
				delayOnTouchOnly: true,
				onEnd: function(event) {
					announcePlace(zone, event.item)
					afterReorder(zone, event.item)
				},
			})
		})

		if (state.refocus !== null) {

			const tile = tilesOf(zone).find((other) => other.dataset.value === state.refocus)

			state.refocus = null

			if (tile !== undefined) {
				tile.focus()
			}
		}
	}

	/** withSortable runs the callback once Sortable is on the page, loading it the first time a widget needs it */
	function withSortable(callback) {

		if (window.Sortable) {
			callback()
			return
		}

		waitingForSortable.push(callback)

		if (document.querySelector('script[src="' + SORTABLE_URL + '"]') !== null) {
			return
		}

		const script = document.createElement("script")
		script.src = SORTABLE_URL
		script.addEventListener("load", function() {
			const callbacks = waitingForSortable
			waitingForSortable = []
			callbacks.forEach((waiting) => waiting())
			window.dispatchEvent(new Event("sortableReady"))
		})
		document.head.appendChild(script)
	}

	/** guardDrop keeps a file dropped outside every widget from replacing the page */
	function guardDrop(event) {

		if (!hasFiles(event) || document.querySelector("[data-upload-multiple]") === null) {
			return
		}

		const target = event.target instanceof Element ? event.target : null

		if (target !== null && target.closest("[data-upload-multiple]") !== null) {
			return
		}

		event.preventDefault()

		if (event.type === "dragover") {
			event.dataTransfer.dropEffect = "none"
		}
	}

	window.addEventListener("dragover", guardDrop)
	window.addEventListener("drop", guardDrop)
	document.addEventListener("submit", guardSubmit, true)
	document.addEventListener("htmx:beforeSend", onFormSending)
	document.addEventListener("htmx:afterRequest", onFormSettled)

	// Every swap can bring a zone, or a row drawn by the step into a zone already on the page
	htmx.onLoad(function(content) {

		if (!(content instanceof Element)) {
			return
		}

		const zones = Array.from(content.querySelectorAll("[data-upload-multiple]"))
		const rows = Array.from(content.querySelectorAll(".upload-multiple-tiles"))

		if (content.matches("[data-upload-multiple]")) {
			zones.unshift(content)
		}

		if (content.matches(".upload-multiple-tiles")) {
			rows.unshift(content)
		}

		zones.forEach(initZone)
		rows.forEach(initRow)
	})

	window.uploadMultiple = true

})();
