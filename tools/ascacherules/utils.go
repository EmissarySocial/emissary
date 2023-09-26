package ascacherules

import "golang.org/x/exp/constraints"

func clamp[T constraints.Ordered](min T, mid T, max T) T {

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
