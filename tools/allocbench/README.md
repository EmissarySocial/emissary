# Allocation Benchmarks

A benchmark-only package that pins how the Go compiler and runtime allocate for a handful of everyday patterns: returning a pointer versus a value, growing a slice with and without preallocated capacity, boxing an int into an interface, and four ways of building a string.

Nothing imports it. It exists so the performance rules in the go-quality skill rest on measurements from this toolchain rather than on folklore, and so a future toolchain that changes one of those answers is noticed. The recorded baseline, and how to compare a new run against it, are in [../../AGENTS.md](../../AGENTS.md).

Run it with `go test -run='^$' -bench=. -benchmem -count=5 ./tools/allocbench/`.
