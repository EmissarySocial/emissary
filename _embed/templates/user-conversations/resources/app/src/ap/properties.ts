type apObject = {
	[key: string]: any
}

export function Id(value: apObject): string {
	return string(value, "id", "ap:id", "https://www.w3.org/ns/activitystreams#id")
}

export function Actor(value: apObject): string {
	return string(value, "actor", "ap:actor", "https://www.w3.org/ns/activitystreams#actor")
}

export function Outbox(value: apObject): string {
	return string(value, "outbox", "ap:outbox", "https://www.w3.org/ns/activitystreams#outbox")
}

export function Type(value: apObject): string {
	return string(value, "type", "ap:type", "https://www.w3.org/ns/activitystreams#type")
}

export function Name(value: apObject): string {
	return string(value, "name", "ap:name", "https://www.w3.org/ns/activitystreams#name")
}

export function Summary(value: apObject): string {
	return string(value, "summary", "ap:summary", "https://www.w3.org/ns/activitystreams#summary")
}

export function Content(value: apObject): string {
	return string(value, "content", "ap:content", "https://www.w3.org/ns/activitystreams#content")
}

export function MlsMessage(value: apObject): string {
	return string(value, "messages", "mls:messages", "https://purl.archive.org/socialweb/mls#messages")
}

export function MlsKeyPackages(value: apObject): string {
	return string(value, "keyPackages", "mls:keyPackages", "https://purl.archive.org/socialweb/mls#keyPackages")
}

export function EventStream(value: apObject): string {
	return string(value, "eventStream", "sse:eventStream", "https://purl.archive.org/socialweb/sse#eventStream")
}

function string(value: apObject, ...names: string[]): string {
	for (const name of names) {
		if (value[name] != undefined) {
			const result = value[name]
			if (typeof result === "string") {
				return result
			}
		}
	}

	return ""
}
