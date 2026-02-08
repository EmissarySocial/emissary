// APActor represents the actor format guaranteed
// to be provided by the Emissary server. ActivityPub actors
// have many other options, but these are the ones we're
// using in this app.
export type APActor = {
	id: string
	name: string
	username: string
	icon: string
	inbox: string
	outbox: string
	"mls:keyPackages": string
	"mls:messages": string
}
