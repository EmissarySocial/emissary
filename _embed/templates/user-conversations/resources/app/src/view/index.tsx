import m from "mithril"
import {type Vnode} from "mithril"
import {type DBGroup} from "../model/db-group"
import {Controller} from "../controller"
import {NewConversation} from "./modal-newConversation"
import {EditGroup} from "./modal-editGroup"

type IndexVnode = Vnode<IndexAttrs, IndexState>

type IndexAttrs = {
	controller: Controller
}

type IndexState = {
	modal: string
	modalGroup?: DBGroup
	groups: DBGroup[]
}

export class Index {
	oninit(vnode: IndexVnode) {
		vnode.state.modal = ""
		vnode.state.groups = []
		this.loadGroups(vnode)
	}

	public view(vnode: IndexVnode) {
		return (
			<div id="conversations">
				<div
					id="conversation-list"
					class="table no-top-border width-50% md:width-40% lg:width-30% flex-shrink-0 scroll-vertical">
					<div
						role="button"
						class="link conversation-selector padding flex-row flex-align-center"
						onclick={() => (vnode.state.modal = "NEW-CONVERSATION")}>
						<div
							class="circle width-32 flex-shrink-0 flex-center margin-none"
							style="font-size:24px;background-color:var(--blue50);color:var(--white);">
							<i class="bi bi-plus"></i>
						</div>
						<div class="ellipsis-block" style="max-height:3em;">
							New Conversation
						</div>
					</div>

					{this.viewGroups(vnode)}
				</div>
				<div id="conversation-details" class="width-75%">
					Here be details...
				</div>

				{this.viewModals(vnode)}
			</div>
		)
	}

	async loadGroups(vnode: IndexVnode) {
		vnode.state.groups = await vnode.attrs.controller.allGroups()
		m.redraw()
	}

	private viewGroups(vnode: IndexVnode): JSX.Element[] {
		return vnode.state.groups.map((group) => (
			<div role="button" class="flex-row flex-align-center padding hover-trigger">
				<span class="width-32 circle flex-center">
					<i class="bi bi-lock-fill"></i>
				</span>
				<span class="flex-grow nowrap ellipsis">
					<div>{group.name}</div>
					<div class="text-xs text-light-gray">{group.id}</div>
				</span>
				<button
					onclick={() => {
						console.log(group)
						this.editGroup(vnode, group)
					}}
					class="hover-show">
					&#8943;
				</button>
			</div>
		))
	}

	private editGroup(vnode: IndexVnode, group: DBGroup) {
		vnode.state.modal = "EDIT-GROUP"
		vnode.state.modalGroup = group
		m.redraw()
	}

	private viewModals(vnode: IndexVnode): JSX.Element | undefined {
		switch (vnode.state.modal) {
			case "NEW-CONVERSATION":
				return (
					<NewConversation
						controller={vnode.attrs.controller}
						close={() => this.closeModal(vnode)}></NewConversation>
				)

			case "EDIT-GROUP":
				return (
					<EditGroup
						controller={vnode.attrs.controller}
						group={vnode.state.modalGroup}
						close={() => this.closeModal(vnode)}></EditGroup>
				)
		}

		return undefined
	}

	// Global Modal Snowball
	closeModal(vnode: IndexVnode) {
		document.getElementById("modal")?.classList.remove("ready")

		window.setTimeout(() => {
			vnode.state.modal = ""
			m.redraw()
		}, 240)
	}
}
