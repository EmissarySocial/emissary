package allocbench

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// user is a small struct, near the size where a value return beats a pointer.
type user struct {
	Name string
	Age  int
}

// Sinks keep benchmark results alive so the compiler cannot discard the work.
var (
	sinkAny    any
	sinkUser   user
	sinkUserPt *user
	sinkString string
	sinkInt    int
)

// parts feeds the loop-concatenation benchmarks from outside the loop.
var parts = []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}

// boxedSmall and boxedLarge straddle the runtime.staticuint64s window.
var (
	boxedSmall = 7
	boxedLarge = 99999
)

// newUserPointer returns a pointer whose referent escapes its frame.
func newUserPointer() *user {
	return &user{Name: "John", Age: 30}
}

// newUserValue returns the same struct by value.
func newUserValue() user {
	return user{Name: "John", Age: 30}
}

// box converts an int to an interface, allocating outside the small-int window.
func box(value int) any {
	return value
}

func BenchmarkReturnPointer(b *testing.B) {
	for b.Loop() {
		sinkUserPt = newUserPointer()
	}
}

func BenchmarkReturnValue(b *testing.B) {
	for b.Loop() {
		sinkUser = newUserValue()
	}
}

func BenchmarkAppendNoPrealloc(b *testing.B) {
	for b.Loop() {
		var ids []int
		for i := range 1000 {
			ids = append(ids, i)
		}
		sinkInt = len(ids)
	}
}

func BenchmarkAppendPrealloc(b *testing.B) {
	for b.Loop() {
		ids := make([]int, 0, 1000)
		for i := range 1000 {
			ids = append(ids, i)
		}
		sinkInt = len(ids)
	}
}

func BenchmarkBoxSmallInt(b *testing.B) {
	for b.Loop() {
		sinkAny = box(boxedSmall)
	}
}

func BenchmarkBoxLargeInt(b *testing.B) {
	for b.Loop() {
		sinkAny = box(boxedLarge)
	}
}

func BenchmarkFixedConcat(b *testing.B) {
	for b.Loop() {
		sinkString = "Name: " + parts[0] + ", Age: " + strconv.Itoa(boxedLarge)
	}
}

func BenchmarkFixedBuilder(b *testing.B) {
	for b.Loop() {
		var builder strings.Builder
		builder.WriteString("Name: ")
		builder.WriteString(parts[0])
		builder.WriteString(", Age: ")
		builder.WriteString(strconv.Itoa(boxedLarge))
		sinkString = builder.String()
	}
}

func BenchmarkFixedSprintf(b *testing.B) {
	for b.Loop() {
		sinkString = fmt.Sprintf("Name: %s, Age: %d", parts[0], boxedLarge)
	}
}

func BenchmarkLoopConcat(b *testing.B) {
	for b.Loop() {
		result := ""
		for _, part := range parts {
			result += part + ","
		}
		sinkString = result
	}
}

func BenchmarkLoopBuilder(b *testing.B) {
	for b.Loop() {
		var builder strings.Builder
		for _, part := range parts {
			builder.WriteString(part)
			builder.WriteString(",")
		}
		sinkString = builder.String()
	}
}

func BenchmarkLoopBuilderGrow(b *testing.B) {
	for b.Loop() {
		var builder strings.Builder
		builder.Grow(64)
		for _, part := range parts {
			builder.WriteString(part)
			builder.WriteString(",")
		}
		sinkString = builder.String()
	}
}
