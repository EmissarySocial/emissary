import m from "mithril"
import {type Vnode} from "mithril"
import {Controller} from "../controller"
import {NewConversation} from "./modal-newConversation"

type IndexVnode = Vnode<IndexAttrs, IndexState>

type IndexAttrs = {
	controller: Controller
}

type IndexState = {
	modal: string
}

export class Index {
	oninit(vnode: IndexVnode) {
		vnode.state.modal = ""
	}

	view(vnode: IndexVnode) {
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

					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<img class="circle width-32" />
						<span class="flex-grow nowrap ellipsis">Direct Message 1</span>
						<button class="hover-show">&#8943;</button>
					</div>
					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<span class="width-32 circle flex-center">
							<i class="bi bi-lock-fill"></i>
						</span>
						<span class="flex-grow nowrap ellipsis">Encrypted Conversation</span>
						<button class="hover-show">&#8943;</button>
					</div>
					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<span class="width-32 circle flex-center">
							<i class="bi bi-lock-fill"></i>
						</span>
						<span class="flex-grow nowrap ellipsis">Encrypted Conversation</span>
						<button class="hover-show">&#8943;</button>
					</div>
				</div>
				<div id="conversation-details" class="width-75%">
					Here be details...
				</div>

				<NewConversation
					controller={vnode.attrs.controller}
					modal={vnode.state.modal}
					close={() => this.closeModal(vnode)}></NewConversation>
			</div>
		)
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
