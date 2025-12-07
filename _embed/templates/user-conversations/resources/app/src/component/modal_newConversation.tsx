import  m from "mithril";
import {type Vnode, type VnodeDOM, type Component } from "mithril";
import {Modal} from "./modal"

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
			<div>Hello World!</div>
			<div>
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
}