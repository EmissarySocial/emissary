package first

func String(values ...string) string {
	for index := range values {
		if values[index] != "" {
			return values[index]
		}
	}
	return ""
}

func Int(values ...int) int {
	for index := range values {
		if values[index] != 0 {
			return values[index]
		}
	}
	return 0
}
