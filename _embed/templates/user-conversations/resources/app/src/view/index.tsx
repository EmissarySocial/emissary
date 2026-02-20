import m from "mithril"
import {type Vnode} from "mithril"
import {type Group} from "../model/group"
import {Controller} from "../controller"
import {NewConversation} from "./modal-newConversation"
import {EditGroup} from "./modal-editGroup"
import {WidgetMessageCreate} from "./widget-message-create"
import {Debug} from "./modal-debug"

type IndexVnode = Vnode<IndexAttrs, IndexState>

type IndexAttrs = {
	controller: Controller
}

type IndexState = {
	modal: string
	modalGroup?: Group
}

export class Index {
	oninit(vnode: IndexVnode) {
		vnode.state.modal = ""
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
						onclick={() => this.newConversation(vnode)}>
						<div
							class="circle width-32 flex-shrink-0 flex-center margin-none"
							style="font-size:24px;background-color:var(--blue50);color:var(--white);">
							<i class="bi bi-plus"></i>
						</div>
						<div class="ellipsis-block" style="max-height:3em;">
							Start a Conversation
						</div>
					</div>

					{this.viewGroups(vnode)}
				</div>
				<div id="conversation-details" class="width-75%">
					{this.viewMessages(vnode)}
				</div>

				{this.viewModals(vnode)}
			</div>
		)
	}

	private viewGroups(vnode: IndexVnode): JSX.Element[] {
		const controller = vnode.attrs.controller
		const groups = controller.groups()
		const selectedGroupId = controller.selectedGroupId

		return groups.map((group) => {
			var cssClass = "flex-row flex-align-center padding hover-trigger"

			if (group.id == selectedGroupId) {
				cssClass += " selected"
			}

			return (
				<div role="button" class={cssClass} onclick={() => controller.selectGroup(group.id)}>
					<span class="width-32 circle flex-center">
						<i class="bi bi-lock-fill"></i>
					</span>
					<span class="flex-grow nowrap ellipsis">
						<div>{group.name}</div>
						<div class="text-xs text-light-gray">{group.id}</div>
					</span>
					<button onclick={() => this.editGroup(vnode, group)} class="hover-show">
						&#8943;
					</button>
				</div>
			)
		})
	}

	// viewMessages returns the JSX for the messages within the selectedGroup.
	// If there is no selected group, then a welcome message is shown instead.
	private viewMessages(vnode: IndexVnode): JSX.Element[] {
		//
		// If there's no selected group, then show a welcome message
		if (vnode.attrs.controller.selectedGroupId == "") {
			return [
				<div class="flex-center height-100% align-center">
					<div>
						<div class="margin-vertical bold">Welcome to Conversations!</div>
						<div class="margin-vertical">Messages will appear here once you get started.</div>
						<div class="margin-vertical link" onclick={() => this.newConversation(vnode)}>
							Start a conversation
						</div>
					</div>
				</div>,
			]
		}

		// Otherwise, list messages for the selected group
		const messages = vnode.attrs.controller.messages()

		// Display messages
		return [
			<div class="flex-grow padding-lg">
				{messages.map((message) => {
					return (
						<div class="card padding margin-bottom">
							{message.plaintext}
							<br />
							<div class="text-xs text-light-gray">{message.sender}</div>
						</div>
					)
				})}
			</div>,
			<WidgetMessageCreate controller={vnode.attrs.controller}></WidgetMessageCreate>,
		]
	}

	private newConversation(vnode: IndexVnode) {
		vnode.state.modal = "NEW-CONVERSATION"
	}

	// editGroup opens the "Edit Group" modal for the specified group
	private editGroup(vnode: IndexVnode, group: Group) {
		vnode.state.modal = "EDIT-GROUP"
		vnode.state.modalGroup = group
	}

	// viewModals returns the JSX for the currently active modal dialog, or undefined if no modal is active
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

			case "DEBUG":
				return <Debug controller={vnode.attrs.controller} close={() => this.closeModal(vnode)}></Debug>
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
