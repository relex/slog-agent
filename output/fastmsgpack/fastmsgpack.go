// Package fastmsgpack offers a subset of msgpack serialization operated on fixed length []byte with no heap allocation,
// no IO abstraction and all calls are inlined.
//
// Benchmark indicates at least 100% improvement in an independent worker (in/out are channels).
//
// The calls should only be used for hot paths, e.g. serialization of individual log records, and NOT to be used in tests
// to verify anything.
package fastmsgpack
