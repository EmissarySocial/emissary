import m from "mithril"
import {type Vnode} from "mithril"
import {Controller} from "../controller"
import {Modal} from "./modal"

type CreateKeysVnode = Vnode<CreateKeysArgs, CreateKeysState>

interface CreateKeysArgs {
	controller: Controller
	modal: string
	close: () => void
}

interface CreateKeysState {
	clientName: string
	password: string
	passwordHint: string
}

export class CreateKeys {
	//

	oninit(vnode: CreateKeysVnode) {
		vnode.state.clientName = this.defaultClientName()
		vnode.state.password = ""
		vnode.state.passwordHint = ""
	}

	view(vnode: CreateKeysVnode) {
		// RULE: Only display when state is correct
		if (vnode.attrs.modal != "SETUP-KEYS") {
			return null
		}

		return (
			<Modal close={vnode.attrs.close}>
				<form onsubmit={(event: SubmitEvent) => this.onSubmit(event, vnode)}>
					<div class="layout layout-vertical">
						<h1>
							<i class="bi bi-key"></i> Encryption Keys
						</h1>

						<div class="margin-vertical">
							Private Keys are stored only on this device and never shared with anyone. Choose a password
							to lock your private keys on this device.
						</div>

						<div class="margin-vertical">
							<b>BE CAREFUL!</b> If you lose this password, you will not be able to recover your private
							message history, so please store your password in a safe place, such as a password manager.
						</div>

						<div class="layout-elements">
							<div class="layout-element">
								<label for="password">Conversation Password</label>
								<input
									type="password"
									id="password"
									name="password"
									required="true"
									autocomplete="new-password"
									value={vnode.state.password}
									oninput={(event: Event) => this.setPassword(vnode, event)}></input>
								<div class="text-sm text-gray">
									Should be different from your account password (which is stored on your server). If
									you lose this password, you will lose your encrypted message history.
								</div>
							</div>
							<div class="layout-element">
								<label for="passwordHint">Password Hint</label>
								<input
									type="text"
									id="passwordHint"
									name="passwordHint"
									value={vnode.state.passwordHint}
									oninput={(event: Event) => this.setPasswordHint(vnode, event)}></input>
								<div class="text-sm text-gray">
									(Optional) Helps you remember your password in case your forget it.
								</div>
							</div>
							<div class="layout-element">
								<label for="clientName">Device Name</label>
								<input
									type="text"
									id="clientName"
									name="clientName"
									value={vnode.state.clientName}
									maxlength="128"
									autocomplete="off"
									data-1p-ignore
									required="true"
									oninput={(event: Event) => this.setClientName(vnode, event)}></input>
								<div class="text-sm text-gray">
									Helps identify this device in the{" "}
									<a href="/@me/settings/keyPackages" target="_blank">
										key manager <i class="bi bi-box-arrow-up-right"></i>
									</a>
								</div>
							</div>
						</div>
					</div>
					<div class="margin-top">
						<button class="primary">Create Encryption Keys</button>
						<button onclick={vnode.attrs.close} tabIndex="0">
							Close
						</button>
					</div>
				</form>
			</Modal>
		)
	}

	setClientName(vnode: CreateKeysVnode, event: Event) {
		const input = event.target as HTMLInputElement
		vnode.state.clientName = input.value
	}

	setPassword(vnode: CreateKeysVnode, event: Event) {
		const input = event.target as HTMLInputElement
		vnode.state.password = input.value
	}

	setPasswordHint(vnode: CreateKeysVnode, event: Event) {
		const input = event.target as HTMLInputElement
		vnode.state.passwordHint = input.value
	}

	async onSubmit(event: SubmitEvent, vnode: CreateKeysVnode) {
		event.preventDefault()
		await vnode.attrs.controller.createEncryptionKeys(
			vnode.state.clientName,
			vnode.state.password,
			vnode.state.passwordHint
		)
		vnode.attrs.close()
	}

	close(vnode: CreateKeysVnode) {
		vnode.attrs.close()
	}

	defaultClientName() {
		const userAgent = navigator.userAgent

		var result = "Unknown Browser"

		// Estimate the Browser Name
		if (userAgent.indexOf("Edge") != -1) {
			result = "Microsoft Edge"
		} else if (userAgent.indexOf("Chrome") != -1) {
			result = "Google Chrome"
		} else if (userAgent.indexOf("Firefox") != -1) {
			result = "Mozilla Firefox"
		} else if (userAgent.indexOf("Safari") != -1) {
			result = "Apple Safari"
		} else if (userAgent.indexOf("Opera") != -1) {
			result = "Opera"
		} else if (userAgent.indexOf("Vivaldi") != -1) {
			result = "Vivaldi"
		}

		// Estimate the OS Name
		if (userAgent.indexOf("Macintosh") != -1) {
			result += " on Macintosh"
		} else if (userAgent.indexOf("Windows") != -1) {
			result += " on Windows"
		} else if (userAgent.indexOf("Linux") != -1) {
			result += " on Linux"
		} else if (userAgent.indexOf("Android") != -1) {
			result += " on Android"
		} else if (userAgent.indexOf("iPhone") != -1) {
			result += " on iOS"
		} else if (userAgent.indexOf("iPad") != -1) {
			result += " on iPadOS"
		} else {
			result += " on Unknown OS"
		}

		return result
	}
}
