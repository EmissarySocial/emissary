/**
 * This file is the browser side of the single-file zone that the form library's upload widget draws.
 *
 * The widget draws the zone and everything inside it: the saved file as a tile, when there is one,
 * the plus, the file input, and the status line.  The file travels with the form's Save, so this
 * script uploads nothing.  It adds what the widget cannot: opening the picker from the plus or the
 * tile, taking a dropped file into the input, drawing the picked file as a tile before it is saved,
 * and putting the saved file back when the pick is removed.
 *
 * The input is the only state.  Every change ends in render, which draws the zone from the input's
 * files, so a pick, a drop, and a removal cannot disagree about what the Save will carry.
 */

(function() {

	if (window.uploadSingle) {
		return
	}

	/** depths holds each zone's count of drag enters minus drag leaves, so a child's dragleave does not clear the highlight */
	const depths = new WeakMap()

	/** inputOf returns the zone's file input */
	function inputOf(zone) {
		return zone.querySelector("input[type=file]")
	}

	/** addOf returns the zone's plus */
	function addOf(zone) {
		return zone.querySelector(".upload-multiple-add")
	}

	/** savedOf returns the tile of the file the server already holds, or null */
	function savedOf(zone) {
		return zone.querySelector("[data-upload-single-saved]")
	}

	/** pickedOf returns the tile of the file picked but not saved yet, or null */
	function pickedOf(zone) {
		return zone.querySelector("[data-upload-single-picked]")
	}

	/** announce writes the zone's status line, in the warning color when it reports a refusal */
	function announce(zone, message, isWarning) {

		const status = zone.querySelector(".upload-multiple-status")

		if (status === null) {
			return
		}

		status.textContent = message
		status.classList.toggle("warning", isWarning === true)
	}

	/** hasFiles reports whether a drag carries files from outside the page */
	function hasFiles(event) {
		return event.dataTransfer !== null && event.dataTransfer !== undefined && Array.from(event.dataTransfer.types).includes("Files")
	}

	/** accepts reports whether the input's accept attribute allows a file, as the picker would */
	function accepts(input, file) {

		const patterns = (input.getAttribute("accept") || "").split(",").map((pattern) => pattern.trim().toLowerCase()).filter((pattern) => pattern !== "")

		if (patterns.length === 0) {
			return true
		}

		const type = (file.type || "").toLowerCase()
		const name = (file.name || "").toLowerCase()

		return patterns.some(function(pattern) {

			if (pattern.startsWith(".")) {
				return name.endsWith(pattern)
			}

			if (pattern.endsWith("/*")) {
				return type.startsWith(pattern.slice(0, -1))
			}

			return type === pattern
		})
	}

	/** drawIcon fills a tile with the icon and the name that stand in for a file with no preview */
	function drawIcon(tile, name) {

		const icon = document.createElement("i")
		icon.className = "bi bi-file-earmark upload-multiple-icon"

		const label = document.createElement("span")
		label.className = "upload-multiple-filename"
		label.textContent = name

		tile.prepend(icon, label)
	}

	/** drawPicked returns the tile for a file that is picked but not saved yet, with the X that removes the pick */
	function drawPicked(file) {

		const tile = document.createElement("div")
		tile.className = "upload-multiple-tile hover-trigger"
		tile.dataset.uploadSinglePicked = "true"
		tile.tabIndex = 0
		tile.setAttribute("aria-label", "Replace " + file.name)

		const remove = document.createElement("button")
		remove.type = "button"
		remove.className = "upload-multiple-remove hover-show"
		remove.setAttribute("aria-label", "Remove " + file.name)

		const glyph = document.createElement("i")
		glyph.className = "bi bi-x-lg"
		remove.appendChild(glyph)
		tile.appendChild(remove)

		if (!(file.type || "").startsWith("image/")) {
			drawIcon(tile, file.name)
			return tile
		}

		// RULE: An image the browser cannot decode falls back to the icon, never to a broken-image mark
		const preview = document.createElement("img")
		preview.className = "upload-multiple-preview"
		preview.alt = file.name
		preview.draggable = false
		preview.addEventListener("error", function() {
			URL.revokeObjectURL(preview.src)
			preview.remove()
			drawIcon(tile, file.name)
		})
		preview.src = URL.createObjectURL(file)
		tile.prepend(preview)

		return tile
	}

	/** removePicked takes the picked file's tile out of the zone and releases its preview */
	function removePicked(zone) {

		const tile = pickedOf(zone)

		if (tile === null) {
			return
		}

		const preview = tile.querySelector(".upload-multiple-preview")

		if (preview !== null && preview.src.startsWith("blob:")) {
			URL.revokeObjectURL(preview.src)
		}

		tile.remove()
	}

	/** render draws the zone from the input's files: the picked file, or else the saved one, or else the plus */
	function render(zone) {

		const input = inputOf(zone)
		const add = addOf(zone)
		const saved = savedOf(zone)

		if (input === null || add === null) {
			return
		}

		removePicked(zone)

		const files = Array.from(input.files || [])

		// With nothing picked, the saved file shows again, or the plus when there is none
		if (files.length === 0) {
			if (saved !== null) {
				saved.hidden = false
			}
			add.hidden = saved !== null
			announce(zone, "")
			return
		}

		// The picked file stands in for the saved one until the Save, or until its X
		if (saved !== null) {
			saved.hidden = true
		}

		add.hidden = true
		add.before(drawPicked(files[0]))

		const what = files.length === 1 ? files[0].name : files.length + " files"
		announce(zone, what + " will upload when you save")
	}

	/** take puts dropped files into the input, as the picker would, and refuses what the field does not accept */
	function take(zone, dropped) {

		const input = inputOf(zone)

		if (input === null) {
			return
		}

		const files = Array.from(dropped)
		const fits = files.filter((file) => accepts(input, file))
		const taken = input.multiple ? fits : fits.slice(0, 1)

		if (taken.length === 0) {
			if (files.length > 0) {
				announce(zone, files[0].name + " is not a file this field takes", true)
			}
			return
		}

		const transfer = new DataTransfer()
		taken.forEach((file) => transfer.items.add(file))
		input.files = transfer.files

		render(zone)
		focusTile(zone)
	}

	/** focusTile moves focus to whatever the zone shows now: the picked tile, the saved tile, or the plus */
	function focusTile(zone) {

		const target = pickedOf(zone) || [savedOf(zone), addOf(zone)].find((element) => element !== null && !element.hidden)

		if (target) {
			target.focus()
		}
	}

	/** clear removes the pick, which puts the saved file or the plus back */
	function clear(zone) {

		const input = inputOf(zone)

		if (input === null) {
			return
		}

		input.value = ""
		render(zone)
		focusTile(zone)
	}

	/** openPicker opens the input's file picker */
	function openPicker(zone) {

		const input = inputOf(zone)

		if (input !== null) {
			input.click()
		}
	}

	/** onClick runs the plus, a picked tile's X, and a click on a tile; a saved tile's X is htmx's own request */
	function onClick(zone, event) {

		const target = event.target instanceof Element ? event.target : null

		if (target === null) {
			return
		}

		const button = target.closest("button")

		if (button !== null && zone.contains(button)) {

			if (button.classList.contains("upload-multiple-add")) {
				openPicker(zone)
				return
			}

			if (button.classList.contains("upload-multiple-remove") && button.closest("[data-upload-single-picked]") !== null) {
				clear(zone)
			}

			return
		}

		if (target.closest(".upload-multiple-tile") !== null) {
			openPicker(zone)
		}
	}

	/** onKeydown opens the picker for Enter or Space on a focused tile, as a button would */
	function onKeydown(zone, event) {

		const tile = event.target instanceof Element && event.target.matches(".upload-multiple-tile") ? event.target : null

		if (tile === null || (event.key !== "Enter" && event.key !== " ")) {
			return
		}

		event.preventDefault()
		openPicker(zone)
	}

	/** onDeleted removes the saved tile once its X's request has gone through, and shows the plus */
	function onDeleted(zone, event) {

		const saved = savedOf(zone)
		const source = event.detail !== undefined && event.detail !== null ? event.detail.elt : null

		if (saved === null || !(source instanceof Element) || !saved.contains(source) || event.detail.successful !== true) {
			return
		}

		saved.remove()
		render(zone)
		focusTile(zone)
	}

	/** initZone wires a zone once: its drag and drop, its picker, its buttons, and its keyboard */
	function initZone(zone) {

		if (zone.dataset.uploadSingleReady === "true") {
			return
		}

		zone.dataset.uploadSingleReady = "true"
		depths.set(zone, 0)

		zone.addEventListener("dragenter", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			depths.set(zone, depths.get(zone) + 1)
			zone.classList.add("highlight")
		})

		zone.addEventListener("dragover", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			event.dataTransfer.dropEffect = "copy"
		})

		zone.addEventListener("dragleave", function(event) {
			if (!hasFiles(event)) {
				return
			}
			depths.set(zone, Math.max(depths.get(zone) - 1, 0))
			if (depths.get(zone) === 0) {
				zone.classList.remove("highlight")
			}
		})

		zone.addEventListener("drop", function(event) {
			if (!hasFiles(event)) {
				return
			}
			event.preventDefault()
			depths.set(zone, 0)
			zone.classList.remove("highlight")
			take(zone, event.dataTransfer.files)
		})

		zone.addEventListener("change", function(event) {
			if (event.target === inputOf(zone)) {
				render(zone)
			}
		})

		zone.addEventListener("click", (event) => onClick(zone, event))
		zone.addEventListener("keydown", (event) => onKeydown(zone, event))
		zone.addEventListener("htmx:afterRequest", (event) => onDeleted(zone, event))

		// A browser can restore a picked file into the input when the page comes back
		render(zone)
	}

	/** guardDrop keeps a file dropped beside a zone from replacing the page */
	function guardDrop(event) {

		if (!hasFiles(event) || document.querySelector("[data-upload-single]") === null) {
			return
		}

		const target = event.target instanceof Element ? event.target : null

		if (target !== null && target.closest("[data-upload-single], [data-upload-multiple]") !== null) {
			return
		}

		event.preventDefault()

		if (event.type === "dragover") {
			event.dataTransfer.dropEffect = "none"
		}
	}

	window.addEventListener("dragover", guardDrop)
	window.addEventListener("drop", guardDrop)

	// Every swap can bring a zone
	htmx.onLoad(function(content) {

		if (!(content instanceof Element)) {
			return
		}

		const zones = Array.from(content.querySelectorAll("[data-upload-single]"))

		if (content.matches("[data-upload-single]")) {
			zones.unshift(content)
		}

		zones.forEach(initZone)
	})

	window.uploadSingle = true

})();
