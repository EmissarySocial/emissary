import m from "mithril"
import { type ServiceFactory} from "./service/factory"

export class ViewContainer {

	// All class #properties are PRIVATE
	#factory: ServiceFactory

	constructor(factory:ServiceFactory) {
		console.log(factory)
		this.#factory = factory
	}

	public view() {
		return <div class="flex-row">
			<div class="table no-top-border width-25% flex-shrink-0 scroll-vertical" style="background-color:var(--gray10); view-transition-name:second-sidebar;">

				<div
					role="button"
					class="link conversation-selector pos-relative padding-horizontal-sm flex-row flex-align-center">

					<div class="width-32 flex-shrink-0 flex-center">
						<div class="circle width-32 flex-shrink-0 flex-center" style="font-size:24px;background-color:var(--blue50);color:var(--white);">+</div>
					</div>
					<div class="ellipsis-block" style="max-height:3em;">New Conversation</div>
				</div>

				<div>Convo 1</div>
				<div>Convo 2</div>
				<div>Convo 3</div>
			</div>
			<div class="width-75%">
				Here be details...
			</div>
		</div>
	}
}