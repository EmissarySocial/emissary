import  m, { request } from "mithril";
import { type Vnode, type VnodeDOM, type Component } from "mithril";
import { type APActor } from "../model/actor"
import { Modal } from "./modal"
import { keyCode } from "./utils"

type ActorSearchVnode = VnodeDOM<ActorSearchArgs, ActorSearchState>

interface ActorSearchArgs {
	name: string
	endpoint: string
	onselect: (actors:APActor[]) => void
}

interface ActorSearchState {
	search: string
	loading: boolean
	options: APActor[]
	selected: APActor[]
	highlightedOption: number
	encrypted: boolean
}

export class ActorSearch {

	oninit(vnode: ActorSearchVnode) {
		vnode.state.search = ""
		vnode.state.loading = false
		vnode.state.options = []
		vnode.state.selected = []
		vnode.state.highlightedOption = -1
	}

	view(vnode: ActorSearchVnode) {

		return (
			<div class="autocomplete">
				<div class="input">
					{vnode.state.selected.map((actor, index) => {
						const isSecure = actor.keyPackages != ""
						return <span class={isSecure ? "tag blue" : "tag gray"}>
							<span style="display:inline-flex; align-items:center; margin-right:8px;">
								<img src={actor.icon} class="circle" style="height:1em; margin:0px 4px;"/>
								<span class="bold">{actor.name}</span>
								&nbsp;
								{ isSecure ? <i class="bi bi-lock-fill"></i> : null}
							</span>
							<i class="clickable bi bi-x-lg" onclick={()=> this.removeActor(vnode, index)}></i>
						</span>
					})}
					<input 
						id="idActorSearch"
						name={vnode.attrs.name} 
						class="padding-none"
						style="min-width:200px;"
						value={vnode.state.search}
						tabindex="0"
						onkeydown={async(event:KeyboardEvent)=>{
							this.onkeydown(event, vnode)
						}}
						onkeypress={async(event:KeyboardEvent)=>{
							this.onkeypress(event, vnode)
						}}
						oninput={async(event:KeyboardEvent)=>{
							this.oninput(event, vnode)
						}}
						onfocus={()=>this.loadOptions(vnode)}
						onblur={()=>this.onblur(vnode)}>
					</input>
				</div>
				{ vnode.state.options.length ? 
					<div class="options">
						<div role="menu" class="menu">
							{vnode.state.options.map((actor, index) => {
								const isSecure = actor.keyPackages != ""
								return <div
									role="menuitem"
									class="flex-row padding-xs"
									onmousedown={()=>this.selectActor(vnode, index)}
									aria-selected={(index == vnode.state.highlightedOption) ? "true" : null }>
									<div class="width-32">
										<img src={actor.icon} class="width-32 circle"/>
									</div>
									<div>
										<div>
											{actor.name} &nbsp;
											{ isSecure ? <i class="text-xs text-light-gray bi bi-lock-fill"></i> : null}
										</div>
										<div class="text-xs text-light-gray">{actor.actorId}</div>
									</div>
								</div>
							})}
						</div>
					</div>
				: null }
			</div>
		)
	}

	async onkeydown(event: KeyboardEvent, vnode: ActorSearchVnode) {

		switch(keyCode(event)) {
		
		case "Backspace":
			const target = event.target as HTMLInputElement

			if (target?.selectionStart == 0) {
				this.removeActor(vnode, vnode.state.selected.length-1)
				event.stopPropagation()
			}
			return

		case "ArrowDown":
			vnode.state.highlightedOption = Math.min(vnode.state.highlightedOption+1, vnode.state.options.length-1)
			return

		case "ArrowUp":
			vnode.state.highlightedOption = Math.max(vnode.state.highlightedOption-1, 0)
			return

		case "Enter":
			this.selectActor(vnode, vnode.state.highlightedOption)
			return
		}
	}

	async onkeypress(event: KeyboardEvent, vnode: ActorSearchVnode) {

		switch(keyCode(event)) {
		
		case "ArrowDown":
			event.stopPropagation()
			return

		case "ArrowUp":
			event.stopPropagation()
			return

		case "Enter":
			event.stopPropagation()
			return

		case "Escape":
			if (vnode.state.options.length > 0) {
				vnode.state.options = []
				event.stopPropagation()
			}
			return
		}

	}

	async oninput(event: KeyboardEvent, vnode: ActorSearchVnode) {
		const target = event.target as HTMLInputElement
		vnode.state.search = target.value
		this.loadOptions(vnode)
	}

	async loadOptions(vnode: ActorSearchVnode) {

		if (vnode.state.search == "") {
			vnode.state.options = []
			vnode.state.highlightedOption = -1
			return
		}

		vnode.state.loading = true
		vnode.state.options = await m.request(vnode.attrs.endpoint + "?q=" + vnode.state.search)
		vnode.state.loading = false
		vnode.state.highlightedOption = -1
	}

	onblur(vnode: ActorSearchVnode) {
		requestAnimationFrame(()=> {
			vnode.state.options = []
			vnode.state.highlightedOption = -1
			m.redraw()
		})
	}

	selectActor(vnode: ActorSearchVnode, index:number) {
		const selected = vnode.state.options[index]

		if (selected == null) {
			return
		}

		vnode.state.selected.push(selected)
		vnode.state.options = []
		vnode.state.search = ""
		vnode.attrs.onselect(vnode.state.selected)
	}

	removeActor(vnode: ActorSearchVnode, index:number) {
		vnode.state.selected.splice(index, 1)
		vnode.attrs.onselect(vnode.state.selected)
		requestAnimationFrame(() =>
			document.getElementById("idActorSearch")?.focus()
		)
	}
}