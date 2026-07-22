package ascacherules

import "cmp"

func clamp[T cmp.Ordered](min T, mid T, max T) T {

	if min > max {
		return max
	}

	if min > mid {
		return min
	}

	if mid > max {
		return max
	}

	return mid
}
