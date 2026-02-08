import m, {request} from "mithril"
import {type Vnode, type VnodeDOM, type Component} from "mithril"
import {type APActor} from "../model/ap-actor"
import {keyCode} from "./utils"
import {type APCollectionHeader} from "../model/ap-collection"

type ActorSearchVnode = VnodeDOM<ActorSearchArgs, ActorSearchState>

interface ActorSearchArgs {
	name: string
	value: APActor[]
	endpoint: string
	onselect: (actors: APActor[]) => void
}

interface ActorSearchState {
	search: string
	loading: boolean
	actors: APActor[]
	keyPackages: {[key: string]: number}
	highlightedOption: number
	encrypted: boolean
}

export class ActorSearch {
	oninit(vnode: ActorSearchVnode) {
		vnode.state.search = ""
		vnode.state.loading = false
		vnode.state.actors = []
		vnode.state.keyPackages = {}
		vnode.state.highlightedOption = -1
	}

	view(vnode: ActorSearchVnode) {
		return (
			<div class="autocomplete">
				<div class="input">
					{vnode.attrs.value.map((actor, index) => {
						const keyPackageCount = vnode.state.keyPackages[actor.id]
						const isSecure = keyPackageCount != undefined && keyPackageCount > 0
						return (
							<span class={isSecure ? "tag blue" : "tag gray"}>
								<span style="display:inline-flex; align-items:center; margin-right:8px;">
									<img src={actor.icon} class="circle" style="height:1em; margin:0px 4px;" />
									<span class="bold">{actor.name}</span>
									&nbsp;
									{isSecure ? <i class="bi bi-lock-fill"></i> : null}
								</span>
								<i class="clickable bi bi-x-lg" onclick={() => this.removeActor(vnode, index)}></i>
							</span>
						)
					})}
					<input
						id="idActorSearch"
						name={vnode.attrs.name}
						class="padding-none"
						style="min-width:200px;"
						value={vnode.state.search}
						tabindex="0"
						onkeydown={async (event: KeyboardEvent) => {
							this.onkeydown(event, vnode)
						}}
						onkeypress={async (event: KeyboardEvent) => {
							this.onkeypress(event, vnode)
						}}
						oninput={async (event: KeyboardEvent) => {
							this.oninput(event, vnode)
						}}
						onfocus={() => this.loadOptions(vnode)}
						onblur={() => this.onblur(vnode)}></input>
				</div>
				{vnode.state.actors.length ? (
					<div class="options">
						<div role="menu" class="menu">
							{vnode.state.actors.map((actor, index) => {
								const keyPackageCount = vnode.state.keyPackages[actor.id]
								const isSecure = keyPackageCount != undefined && keyPackageCount > 0
								return (
									<div
										role="menuitem"
										class="flex-row padding-xs"
										onmousedown={() => this.selectActor(vnode, index)}
										aria-selected={index == vnode.state.highlightedOption ? "true" : null}>
										<div class="width-32">
											<img src={actor.icon} class="width-32 circle" />
										</div>
										<div>
											<div>
												{actor.name} &nbsp;
												{isSecure ? (
													<i class="text-xs text-light-gray bi bi-lock-fill"></i>
												) : null}
											</div>
											<div class="text-xs text-light-gray">{actor.username}</div>
										</div>
									</div>
								)
							})}
						</div>
					</div>
				) : null}
			</div>
		)
	}

	async onkeydown(event: KeyboardEvent, vnode: ActorSearchVnode) {
		switch (keyCode(event)) {
			case "Backspace":
				const target = event.target as HTMLInputElement

				if (target?.selectionStart == 0) {
					this.removeActor(vnode, vnode.attrs.value.length - 1)
					event.stopPropagation()
				}
				return

			case "ArrowDown":
				vnode.state.highlightedOption = Math.min(
					vnode.state.highlightedOption + 1,
					vnode.state.actors.length - 1,
				)
				return

			case "ArrowUp":
				vnode.state.highlightedOption = Math.max(vnode.state.highlightedOption - 1, 0)
				return

			case "Enter":
				this.selectActor(vnode, vnode.state.highlightedOption)
				return
		}
	}

	// These event handlers prevent default behavior for certain control keys
	async onkeypress(event: KeyboardEvent, vnode: ActorSearchVnode) {
		switch (keyCode(event)) {
			case "ArrowDown":
			case "ArrowUp":
			case "Enter":
				event.stopPropagation()
				event.preventDefault()
				return

			case "Escape":
				if (vnode.state.actors.length > 0) {
					vnode.state.actors = []
				}
				event.stopPropagation()
				event.preventDefault()
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
			vnode.state.actors = []
			vnode.state.highlightedOption = -1
			return
		}

		vnode.state.loading = true
		vnode.state.actors = await m.request(vnode.attrs.endpoint + "?q=" + vnode.state.search)
		vnode.state.loading = false
		vnode.state.highlightedOption = -1

		this.loadKeyPackages(vnode)
	}

	// (async) Maintains a cache that counts the keyPackages for each actor
	loadKeyPackages(vnode: ActorSearchVnode) {
		//
		//
		for (const actor of vnode.state.actors) {
			if (vnode.state.keyPackages[actor.id] == undefined) {
				if (actor["mls:keyPackages"] == null) {
					continue
				}

				if (actor["mls:keyPackages"] == "") {
					continue
				}

				m.request<APCollectionHeader>(
					"/.api/collectionHeader?url=" + encodeURIComponent(actor["mls:keyPackages"]),
				).then((header: APCollectionHeader) => {
					if (header != undefined) {
						if (header.totalItems != undefined) {
							vnode.state.keyPackages[actor.id] = header.totalItems
							m.redraw()
						}
					}
				})
			}
		}
	}

	onblur(vnode: ActorSearchVnode) {
		requestAnimationFrame(() => {
			vnode.state.actors = []
			vnode.state.highlightedOption = -1
			m.redraw()
		})
	}

	selectActor(vnode: ActorSearchVnode, index: number) {
		const selected = vnode.state.actors[index]

		if (selected == null) {
			return
		}

		vnode.attrs.value.push(selected)
		vnode.state.actors = []
		vnode.state.search = ""
		vnode.attrs.onselect(vnode.attrs.value)
	}

	removeActor(vnode: ActorSearchVnode, index: number) {
		vnode.attrs.value.splice(index, 1)
		vnode.attrs.onselect(vnode.attrs.value)
		requestAnimationFrame(() => document.getElementById("idActorSearch")?.focus())
	}
}
