import m, { type ChildArrayOrPrimitive, type Vnode } from "mithril"

interface ModalAttrs {
	close: () => void
}

type ModalVnode = Vnode<ModalAttrs, {}>

// Adapted from: https://mithril-by-examples.js.org/examples/modal-2/#modal.js
export class Modal {

	oncreate(vnode: ModalVnode) {

		// Locate the <aside> tag where we'll mount the modal
		const aside = document.getElementsByTagName("aside").item(0)

		if (aside == null) {
			console.log("Tag <aside> must be defined to render this dialog.")
			return 
		}

		const widget = {
			view: () => 
			<div id="modal">
				<div id="modal-underlay" onclick={vnode.attrs.close}></div>
				<div id="modal-window">
					{vnode.children}
				</div>
			</div>
		}

		// Append a container to the <aside> tag
		m.mount(aside, widget)

		// Wait one tick, then add "ready" to the modal
		requestAnimationFrame(() => document.getElementById("modal")?.classList.add("ready"))
	}

	onbeforeremove(v: ModalVnode) {
		document.getElementById("modal")?.classList.remove("ready")
	}

	onremove(v: ModalVnode) {

		// Locate the <aside> tag where we'll mount the modal
		const aside = document.getElementsByTagName("aside").item(0)

		if (aside == null) {
			console.log("Tag <aside> must be defined to render this dialog.")
			return 
		}

		m.mount(aside, null)
	}

	view(v: ModalVnode) {
	}
}