import {
	type Article, 
	type Audio,
	type Image,
	type Note,
	type Video,
}  from "./objects"

// Activities are the top-level content in MLS messages
// https://swicg.github.io/activitypub-e2ee/mls#activities
export type Activity = {
	context: any
	type: "Create" | "Update" | "Delete" | "Like" | "Announce" | "Undo" | "Read" | "Listen" | "View" | "IntransitiveActivity"
	id: string
	object: Article | Audio | Image | Note | Video
}
