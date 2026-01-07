import m, { type ChildArrayOrPrimitive, type VnodeDOM } from "mithril"
import { keyCode, getFocusElements } from "./utils"

interface ModalAttrs {
	close: () => void
}

type ModalVnode = VnodeDOM<ModalAttrs, {}>

// Adapted from: https://mithril-by-examples.js.org/examples/modal-2/#modal.js
export class Modal {

	oncreate(vnode: ModalVnode) {
		requestAnimationFrame(() => {
			document.getElementById("modal")?.classList.add("ready")
			
			const firstElement = vnode.dom.querySelector("[tabIndex]") as HTMLInputElement 
			firstElement?.focus()

			m.redraw()
		})
	}

	view(vnode: ModalVnode) {
		return (
			<div id="modal" onkeydown={(event:KeyboardEvent)=> this.onkeydown(event, vnode)}>
				<div id="modal-underlay" onclick={vnode.attrs.close}></div>
				<div id="modal-window">
					{vnode.children}
				</div>
			</div>
		)
	}

	onkeydown(event: KeyboardEvent, vnode: ModalVnode) {
		
		switch(keyCode(event)) {

			// Trap tab focus
			case "Tab": {
				const [firstElement, lastElement] = getFocusElements(vnode.dom)

				if (document.activeElement == lastElement) {
					firstElement?.focus()
					event.stopPropagation()
					event.preventDefault()
				}
				return
			}

			// Trap tab focus
			case "Shift+Tab": {
				const [firstElement, lastElement] = getFocusElements(vnode.dom)

				if (document.activeElement == firstElement) {
					lastElement?.focus()
					event.stopPropagation()
					event.preventDefault()
				}
				return
			}

			// Close modal window
			case "Escape": {
				vnode.attrs.close()
				return
			}
		}
	}

	// TODO: Need handlers for TAB, SHIFT+TAB, ESCAPE

}