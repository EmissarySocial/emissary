import m, {type Vnode} from "mithril"
import type {Controller} from "../controller"
import {CreateKeys} from "./modal-createKeys"

type WelcomeVnode = Vnode<WelcomeAttrs, WelcomeState>

type WelcomeAttrs = {
	controller: Controller
}

type WelcomeState = {
	modal: string
}

export class Welcome {
	view(vnode: WelcomeVnode) {
		return (
			<div class="app-content">
				<div class="flex-row flex-align-center width-100%">
					<div class="text-xl bold flex-grow">
						<i class="bi bi-chat-fill"></i> Conversations
					</div>
					<div class="nowrap"></div>
				</div>
				<div class="card padding max-width-640 margin-top">
					<div class="margin-bottom-lg">
						Conversations collect all of your personal messages into a single place, including{" "}
						<b class="nowrap">direct messages</b> (which can be read by server admins) and{" "}
						<b class="nowrap">private messages</b>. (which are encrypted and cannot be read by others).{" "}
						<a href="https://emissary.dev/conversations" target="_blank" class="nowrap">
							Learn More About Conversations <i class="bi bi-box-arrow-up-right"></i>
						</a>
					</div>
					<div class="flex-row flex-align-center margin-vertical">
						<button class="primary" onclick={() => (vnode.state.modal = "SETUP-KEYS")}>
							Create Encryption Keys
						</button>
						<div>to participate in encrypted conversations.</div>
					</div>
					<div class="flex-row flex-align-center margin-vertical">
						<button onclick={() => this.skipEncryptionKeys(vnode)}>Continue Without Keys&nbsp;</button>
						<div>to send/receive unencrypted messages only.</div>
					</div>
				</div>

				<CreateKeys
					controller={vnode.attrs.controller}
					modal={vnode.state.modal}
					close={() => this.closeModal(vnode)}></CreateKeys>
			</div>
		)
	}

	async skipEncryptionKeys(vnode: WelcomeVnode) {
		await vnode.attrs.controller.skipEncryptionKeys()
		this.closeModal(vnode)
	}

	// Global Modal Snowball
	closeModal(vnode: WelcomeVnode) {
		document.getElementById("modal")?.classList.remove("ready")

		window.setTimeout(() => {
			vnode.state.modal = ""
			m.redraw()
		}, 240)
	}
}
