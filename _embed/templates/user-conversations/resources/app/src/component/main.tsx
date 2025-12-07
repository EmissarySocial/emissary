import m from "mithril"
import { type ServiceFactory} from "../service/factory"
import { NewConversation } from "./modal_newConversation"

export class Main {

	modal: string 

	constructor(vnode:any) {
		this.modal = ""
	}

	public view() {

		return (
			<div class="flex-row">
				<div class="table no-top-border width-25% flex-shrink-0 scroll-vertical" style="background-color:var(--gray10);">

					<div
						role="button"
						class="link conversation-selector pos-relative padding-horizontal-sm flex-row flex-align-center"
						onclick={this.showModalNewConversation}>

						<div class="width-32 flex-shrink-0 flex-center">
							<div class="circle width-32 flex-shrink-0 flex-center margin-none" style="font-size:24px;background-color:var(--blue50);color:var(--white);">+</div>
						</div>
						<div class="ellipsis-block" style="max-height:3em;">New Conversation</div>
					</div>

					<div>Direct Message 1</div>
					<div>Encrypted (2)</div>
					<div>Encrypted (3)</div>
				</div>
				<div class="width-75%">
					Here be details... [{this.modal}]
				</div>

				{/* Modal Dialog Boxes */}
				<NewConversation modal={this.modal} close={this.closeModal}></NewConversation>

			</div>
		)
	}

	// State Changes

	showModalNewConversation = () => {
		this.modal = "NEW-CONVERSATION"
	}

	closeModal = () => {
		document.getElementById("modal")?.classList.remove("ready")
	
		window.setTimeout(() => {
			this.modal = ""
			m.redraw()
		}, 240)
	}
}