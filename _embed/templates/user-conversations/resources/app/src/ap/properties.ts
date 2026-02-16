type apObject = {
	[key: string]: any
}

export function Id(value: apObject): string {
	return string(value, "id", "@id", "https://www.w3.org/ns/activitystreams#id")
}

export function Outbox(value: apObject): string {
	return string(value, "outbox", "ap:outbox", "https://www.w3.org/ns/activitystreams#outbox")
}

export function Content(value: apObject): string {
	return string(value, "content", "ap:content", "https://www.w3.org/ns/activitystreams#content")
}

export function Type(value: apObject): string {
	return string(value, "type", "@type", "https://www.w3.org/ns/activitystreams#type")
}

export function Name(value: apObject): string {
	return string(value, "name", "ap:name", "https://www.w3.org/ns/activitystreams#name")
}

export function MlsMessage(value: apObject): string {
	return string(value, "messages", "mls:messages", "https://purl.archive.org/socialweb/mls#messages")
}

export function MlsKeyPackages(value: apObject): string {
	return string(value, "keyPackages", "mls:keyPackages", "https://purl.archive.org/socialweb/mls#keyPackages")
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
