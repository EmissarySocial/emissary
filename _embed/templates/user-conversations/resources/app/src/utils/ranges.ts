// rangeToArray consumes all values from a generator and returns them as an array
export function rangeToArray<T>(generator: Generator<T>): T[] {
	var result = []
	for (let value of generator) {
		result.push(value)
	}
	return result
}
