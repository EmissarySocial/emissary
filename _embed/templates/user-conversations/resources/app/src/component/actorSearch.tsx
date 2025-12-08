import  m from "mithril";
import {type Vnode, type VnodeDOM, type Component } from "mithril";
import {Modal} from "./modal"

type ActorSearchVnode = Vnode<ActorSearchArgs, {}>

interface ActorSearchArgs {
	name: string
	endpoint: string
}

export class ActorSearch {

	view(vnode: ActorSearchVnode) {

		const {name} = vnode.attrs

		return (

			<div class="pos-relative">
				<div class="input">
					<input name={name} onkeyup={(evt:KeyboardEvent)=>{this.search(evt)}} style="border:none; field-sizing:content" />
				</div>

				<div class="pos-absolute padding-sm width-100%" style="border:solid 1px var(--gray40); background-color:var(--gray10)">
					Search results will go here...
				</div>
			</div>
		)
	}

	search(event: KeyboardEvent) {
		console.log(event?.target)
	}
}