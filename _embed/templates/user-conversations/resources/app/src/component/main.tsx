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
			<div class="flex-row flex-grow">
				<div class="table no-top-border width-50% md:width-40% lg:width-30% flex-shrink-0 scroll-vertical" style="background-color:var(--gray10);">

					<div
						role="button"
						class="link conversation-selector padding flex-row flex-align-center"
						onclick={() => this.showModal('NEW-CONVERSATION')}>

						<div class="circle width-32 flex-shrink-0 flex-center margin-none" style="font-size:24px;background-color:var(--blue50);color:var(--white);"><i class="bi bi-plus"></i></div>
						<div class="ellipsis-block" style="max-height:3em;">New Conversation</div>
					</div>

					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<img class="circle width-32"/> 
						<span class="flex-grow nowrap ellipsis">Direct Message 1</span>
						<button class="text-xs hover-show">&hellip;</button>
					</div>
					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<span class="width-32 circle flex-center"><i class="bi bi-lock-fill"></i></span>
						<span class="flex-grow nowrap ellipsis">Encrypted Conversation</span>
						<button class="text-xs hover-show">&hellip;</button>
					</div>
					<div role="button" class="flex-row flex-align-center padding hover-trigger">
						<span class="width-32 circle flex-center"><i class="bi bi-lock-fill"></i></span>
						<span class="flex-grow nowrap ellipsis">Encrypted Conversation</span>
						<button class="text-xs hover-show">&hellip;</button>
					</div>
				</div>
				<div class="width-75%">
					Here be details...
				</div>

				{/* Modal Dialog Boxes */}
				<NewConversation modal={this.modal} close={() => this.closeModal()}></NewConversation>

			</div>
		)
	}

	showModal(name:string) {
		this.modal = name
	}

	// Global Modal Snowball
	closeModal() {
		document.getElementById("modal")?.classList.remove("ready")
	
		window.setTimeout(() => {
			this.modal = ""
			m.redraw()
		}, 240)
	}
}