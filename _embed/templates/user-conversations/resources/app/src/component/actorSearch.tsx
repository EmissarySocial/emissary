import  m from "mithril";
import { type Vnode, type VnodeDOM, type Component } from "mithril";
import { type APActor } from "../model/actor"
import { Modal } from "./modal"

type ActorSearchVnode = Vnode<ActorSearchArgs, ActorSearchState>

interface ActorSearchArgs {
	name: string
	endpoint: string
}

interface ActorSearchState {
	actors: APActor[]
}

export class ActorSearch {

	oninit(vnode: ActorSearchVnode) {
		vnode.state.actors = []
	}

	view(vnode: ActorSearchVnode) {

		return (
			<div class="pos-relative">
				<div class="input">
					<input 
						name={vnode.attrs.name} 
						class="padding-none"
						style="border:none; field-sizing:content" 
						autofocus
						onkeyup={(evt:KeyboardEvent)=>{
							// hey howdy.
							const target = evt.target as HTMLInputElement
							m.request("/.api/actors?q=" + target.value)
							.then((actors) => {
								vnode.state.actors = actors as APActor[]
							})
						}}>
					</input>
				</div>
				{ vnode.state.actors.length ? 
					<div role="menu" style="position:absolute; width:100%; border:solid 1px black; background-color:white; padding:4px;">
						{vnode.state.actors.map((value) => 
							<div role="menuitem">{value.name}</div>
						)}
					</div>
				: null }
			</div>
		)
	}
}