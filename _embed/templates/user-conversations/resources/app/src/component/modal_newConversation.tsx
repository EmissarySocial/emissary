import  m from "mithril";
import { type APActor } from "../model/actor";
import {type Vnode, type VnodeDOM, type Component } from "mithril";
import {Modal} from "./modal"
import {ActorSearch} from "./actorSearch"

type NewConversationVnode = Vnode<NewConversationArgs, NewConversationState>

interface NewConversationArgs {
	modal: string
	close: () => void
}

interface NewConversationState {
	actors: APActor[]
	encrypted: boolean
}


export class NewConversation {

	oninit(vnode: NewConversationVnode) {
		vnode.state.actors = []
	}

	view(vnode: NewConversationVnode) {

		// RULE: Only display when state is correct
		if (vnode.attrs.modal != "NEW-CONVERSATION") {
			return null
		}

		return (
		<Modal close={vnode.attrs.close}>

			<div class="layout layout-vertical">
				{this.header(vnode)}
				<div class="layout-elements">
					<div class="layout-element">
						<label for="">Participants</label>
						<ActorSearch name="actorIds" endpoint="/.api/actors" onselect={(actors:APActor[])=>this.selectActors(vnode, actors)}></ActorSearch>
					</div>
					<div class="layout-element">
						<label>Message</label>
						<textarea rows="8"></textarea>
						<div class="text-sm text-gray">{this.description(vnode)}</div>
					</div>
				</div>
			</div>
			<div class="margin-top">
				{this.submitButton(vnode)}
				<button onclick={vnode.attrs.close} tabIndex="0">Close</button>
			</div>

		</Modal>)		
	}

	header(vnode: NewConversationVnode): JSX.Element {

		if (vnode.state.actors.length == 0) {
			return <div class="layout-title"><i class="bi bi-plus"></i> Start a Conversation</div>
		}

		if (vnode.state.encrypted) {
			return <div class="layout-title"><i class="bi bi-shield-lock"></i> Encrypted Message</div>
		}

		return <div class="layout-title"><i class="bi bi-envelope-open"></i> Direct Message</div>
	}

	description(vnode:NewConversationVnode): JSX.Element {
		if (vnode.state.actors.length == 0)  {
			return <span></span>
		}

		if (vnode.state.encrypted) {
			return <div>This will be encrypted before it leaves this device, and will not be readable by anyone other than the recipients.</div>
		}

		return <div><i class="bi bi-exclamation-triangle-fill"></i> One or more of your recipients cannot receive encrypted messages. Others on the Internet may be able to read this message.</div>
	}

	submitButton(vnode: NewConversationVnode): JSX.Element {

		if (vnode.state.actors.length == 0) {
			return <button class="primary" disabled>Start a Conversation</button>
		}

		if (vnode.state.encrypted) {
			return <button class="primary" tabindex="0" onclick={(event:MouseEvent)=>this.onsubmit(vnode)}><i class="bi bi-lock"></i> Send Encrypted</button>
		}

		return <button class="selected" disabled>Send Direct Message</button>
		// return <button class="selected" tabindex="0" onclick={(event:MouseEvent)=>this.onsubmit(vnode)}>Send Direct Message</button>
	}

	selectActors(vnode: NewConversationVnode, actors:APActor[]) {
		vnode.state.actors = actors

		if (actors.some((actor)=>actor.keyPackages == "")) {
			vnode.state.encrypted = false
		} else {
			vnode.state.encrypted = true
		}
	}

	onsubmit(vnode: NewConversationVnode) {
		return () => {
			vnode.attrs.close()
		}
	}
}
