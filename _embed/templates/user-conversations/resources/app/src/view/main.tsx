import m from "mithril"
import stream from "mithril/stream"
import {type Vnode} from "mithril"
import {Controller} from "../controller"
import type {Config} from "../model/config"
import {Welcome} from "./welcome"
import {Index} from "."

type MainVnode = Vnode<MainAttrs, MainState>

type MainAttrs = {
	controller: Controller
}

type MainState = {
	modal: string
	config: Config
}

export class Main {
	oninit(vnode: MainVnode) {
		vnode.state.modal = ""
	}

	view(vnode: MainVnode) {
		const controller = vnode.attrs.controller

		if (!controller.config.ready) {
			return <div class="app-content">Loading...</div>
		}

		if (!controller.config.welcome) {
			return <Welcome controller={controller} />
		}

		return <Index controller={controller} />
	}
}
