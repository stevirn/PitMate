//go:build !windows

// On non-Windows platforms there is no LMU and no shared memory to read, so the
// reader is a stub. This lets the whole project build and be developed on Linux
// or macOS; the adapter just reports "not connected".
package lmu

import "errors"

// errUnsupported explains why the adapter cannot read data off Windows.
var errUnsupported = errors.New("lmu: shared-memory telemetry is only available on Windows")

// newReader always fails off Windows.
func newReader() (reader, error) {
	return nil, errUnsupported
}
