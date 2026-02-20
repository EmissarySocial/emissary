import m, {type Vnode} from "mithril"
import type {Controller} from "../controller"

type WidgetMessageCreateVnode = Vnode<WidgetMessageCreateAttrs, WidgetMessageCreateState>

type WidgetMessageCreateAttrs = {
	controller: Controller
}

type WidgetMessageCreateState = {
	message: string
}

export class WidgetMessageCreate {
	oninit(vnode: WidgetMessageCreateVnode) {
		vnode.state.message = ""
	}

	view(vnode: WidgetMessageCreateVnode) {
		return (
			<div class="input flex-row" style="height:200px;">
				<textarea
					value={vnode.state.message}
					style="border:none; height:100%; resize:none;"
					oninput={(e: Event) => this.oninput(vnode, e)}></textarea>
				<button onclick={() => this.sendMessage(vnode)} disabled={vnode.state.message.trim() === ""}>
					Send
				</button>
			</div>
		)
	}

	oninput(vnode: WidgetMessageCreateVnode, event: Event) {
		const target = event.target as HTMLTextAreaElement
		vnode.state.message = target.value
	}

	sendMessage(vnode: WidgetMessageCreateVnode) {
		if (vnode.state.message.trim() === "") {
			return
		}

		vnode.attrs.controller.sendMessage(vnode.state.message)
		vnode.state.message = ""
	}
}
