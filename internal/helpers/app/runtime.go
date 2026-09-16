package app

import "runtime"

// StackTrace returns the current goroutine's stack trace. Used by panic recovery
// blocks so a recovered panic is logged with enough context to debug.
func StackTrace() string {
	const size = 64 << 10 // enough for 64KB stack traces
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	return string(buf)
}
