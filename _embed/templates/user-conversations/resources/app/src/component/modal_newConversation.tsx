import  m from "mithril";
import {type Vnode, type VnodeDOM, type Component } from "mithril";
import {Modal} from "./modal"
import {ActorSearch} from "./actorSearch"

type NewConversationVnode = Vnode<NewConversationArgs, {}>

interface NewConversationArgs {
	modal: string
	close: () => void
}

export class NewConversation {

	view(vnode: NewConversationVnode) {

		// RULE: Only display when state is correct
		if (vnode.attrs.modal != "NEW-CONVERSATION") {
			return null
		}

		return (
		<Modal close={vnode.attrs.close}>

			<div class="layout layout-vertical">
				<div class="layout-title">{this.label(vnode)}</div>
				<div class="layout-elements">
					<div class="layout-element">
						<label for="">Participants</label>
						<ActorSearch name="actorIds"></ActorSearch>
					</div>
					<div class="layout-element">
						<label>Message</label>
						<textarea rows="8"></textarea>
						<div class="text-sm text-gray"></div>
					</div>
				</div>
			</div>
			<div class="margin-top">
				<button onclick={this.submit(vnode)} class="primary">Start Conversation</button>
				<button onclick={vnode.attrs.close}>Close</button>
			</div>

		</Modal>)		
	}

	submit(vnode: NewConversationVnode) {
		return () => {
			console.log("submit...")
			vnode.attrs.close()
		}
	}

	label(vnode: NewConversationVnode) {
		return "+ New Conversation"
	}
}