import m from "mithril"
import {Controller} from "../controller"
import {type APActor} from "../model/ap-actor"
import {type Vnode, type VnodeDOM, type Component} from "mithril"
import {Modal} from "./modal"
import {ActorSearch} from "./actorSearch"
import type {DBGroup} from "../model/db-group"

type DebugVnode = Vnode<DebugArgs, DebugState>

interface DebugArgs {
	controller: Controller
	group: DBGroup
	close: () => void
}

interface DebugState {
	name: string
}

export class Debug {
	//

	oninit(vnode: DebugVnode) {
		vnode.state.name = vnode.attrs.group.name
	}

	view(vnode: DebugVnode) {
		return (
			<Modal close={vnode.attrs.close}>
				<form onsubmit={(event: SubmitEvent) => this.onsubmit(event, vnode)}>
					<div class="layout layout-vertical">
						<div class="layout-title">
							<i class="bi bi-lock-fill"></i> Edit Group
						</div>

						<div class="layout-elements">
							<div class="layout-element">
								<label for="idGroupName">Group Name</label>
								<input
									id="idGroupName"
									type="text"
									name="actorIds"
									value={vnode.state.name}
									oninput={(event: Event) => this.setName(vnode, event)}
								/>
							</div>
						</div>
					</div>
					<div class="margin-top flex-row">
						<div class="flex-grow">
							<button type="submit" class="primary" tabindex="0">
								Save Changes
							</button>
							<button onclick={vnode.attrs.close} tabIndex="0">
								Close
							</button>
						</div>
						<div>
							<span
								onclick={() => {
									this.delete(vnode)
								}}
								class="clickable text-red">
								Leave Group
							</span>
						</div>
					</div>
				</form>
			</Modal>
		)
	}

	setName(vnode: DebugVnode, event: Event) {
		const target = event.target as HTMLTextAreaElement
		vnode.state.name = target.value
	}

	async onsubmit(event: SubmitEvent, vnode: DebugVnode) {
		//
		// Halt the form submission to prevent a page reload
		event.preventDefault()
		event.stopPropagation()

		// Copy values from the form into the Group object
		vnode.attrs.group.name = vnode.state.name

		// Save the Group to the database
		await vnode.attrs.controller.saveGroup(vnode.attrs.group)

		// Success. Close the modal dialog and redraw the screen
		return this.close(vnode)
	}

	async delete(vnode: DebugVnode) {
		//
		// Confirm the user's intent
		if (!confirm("Are you sure you want to leave this group? This action cannot be undone.")) {
			return
		}

		// Delete the group
		await vnode.attrs.controller.deleteGroup(vnode.attrs.group.id)

		// Close the modal dialog
		this.close(vnode)
	}

	close(vnode: DebugVnode) {
		vnode.attrs.close()
		m.redraw()
	}
}
