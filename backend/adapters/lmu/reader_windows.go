//go:build windows

// Windows implementation of the shared-memory reader. It opens the plugin's
// named memory-mapped files, maps them read-only into this process, and copies
// out consistent snapshots using the plugin's version-counter protocol.
//
// This file only compiles on Windows; reader_other.go is the stub everywhere
// else. It cannot be exercised on the dev machine (Linux), so it is kept small
// and the risky, logic-heavy translation lives in the platform-independent,
// fully tested mapping.go.
package lmu

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// x/sys/windows exposes CreateFileMapping but not OpenFileMapping, so we bind
// OpenFileMappingW from kernel32 ourselves. We deliberately want OPEN (not
// create): if the game/plugin is not running the open fails, rather than
// silently creating an empty, all-zero mapping that would look like live data.
var (
	modkernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procOpenFileMappingW = modkernel32.NewProc("OpenFileMappingW")
)

// openFileMapping wraps OpenFileMappingW. Returns a valid handle, or an error if
// the named mapping does not exist (e.g. the game is not running).
func openFileMapping(access uint32, inherit bool, name *uint16) (windows.Handle, error) {
	var inheritArg uintptr
	if inherit {
		inheritArg = 1
	}
	r, _, err := procOpenFileMappingW.Call(uintptr(access), inheritArg, uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, err // Call sets err from GetLastError when the call fails
	}
	return windows.Handle(r), nil
}

// mappedBuffer is one opened, memory-mapped plugin buffer. base is the view's
// address. It is a uintptr (not unsafe.Pointer) because it points at
// OS-mapped memory, not the Go heap: the Go GC must not track or move it.
type mappedBuffer struct {
	handle windows.Handle
	base   uintptr
	size   uintptr
}

// winReader holds the two mapped buffers PitMate reads. They are opened lazily
// on the first successful read so the adapter can start before the game does.
type winReader struct {
	tele  *mappedBuffer
	score *mappedBuffer
}

// newReader returns a Windows reader. Opening the shared memory is deferred to
// read() so a not-yet-running game is not treated as a fatal error.
func newReader() (reader, error) { return &winReader{}, nil }

// openMapping opens an existing named mapping created by the plugin and maps a
// read-only view of it. The plugin creates the mapping in the Local namespace,
// so this must run in the same Windows session as the game (it does — same PC).
func openMapping(name string, size uintptr) (*mappedBuffer, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := openFileMapping(windows.FILE_MAP_READ, false, namePtr)
	if err != nil {
		return nil, fmt.Errorf("open mapping %s: %w", name, err)
	}
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, size)
	if err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("map view %s: %w", name, err)
	}
	return &mappedBuffer{handle: h, base: addr, size: size}, nil
}

// free releases the view and handle. Safe on a nil receiver.
func (b *mappedBuffer) free() {
	if b == nil {
		return
	}
	if b.base != 0 {
		windows.UnmapViewOfFile(b.base)
	}
	if b.handle != 0 {
		windows.CloseHandle(b.handle)
	}
}

// read lazily opens the mappings (if not already open) and returns consistent
// snapshots of both buffers. ok is false until both are open and readable.
func (r *winReader) read() (tel rf2Telemetry, sc rf2Scoring, ok bool) {
	if r.tele == nil {
		r.tele, _ = openMapping(mmTelemetryName, unsafe.Sizeof(tel))
	}
	if r.score == nil {
		r.score, _ = openMapping(mmScoringName, unsafe.Sizeof(sc))
	}
	if r.tele == nil || r.score == nil {
		return tel, sc, false // game (or plugin) not running yet
	}

	t, okT := readVersioned[rf2Telemetry](r.tele.base)
	s, okS := readVersioned[rf2Scoring](r.score.base)
	if !okT || !okS {
		return tel, sc, false
	}
	return t, s, true
}

// close releases both mappings.
func (r *winReader) close() error {
	r.tele.free()
	r.score.free()
	r.tele, r.score = nil, nil
	return nil
}

// readVersioned copies a versioned buffer out of shared memory. Every plugin
// write is bracketed by two counters: mVersionUpdateBegin (bumped before the
// write) and mVersionUpdateEnd (bumped after). We read begin, copy the buffer,
// then read end; if they differ the writer was mid-update and we retry. This is
// the same lightweight protocol the reference readers use.
//
// The begin/end counters are the first two uint32 fields of every buffer, hence
// reading them at offsets 0 and 4.
func readVersioned[T any](base uintptr) (T, bool) {
	var out T
	size := unsafe.Sizeof(out)

	// The one unavoidable conversion: base is a valid address returned by
	// MapViewOfFile and stays valid until UnmapViewOfFile. go vet's unsafeptr
	// analyzer cannot know that and flags it; it is correct here. This is the
	// only place in the package that does it.
	p := unsafe.Pointer(base) //nolint:govet // syscall-returned mapped address
	dst := unsafe.Slice((*byte)(unsafe.Pointer(&out)), size)
	src := unsafe.Slice((*byte)(p), size)

	const maxTries = 8
	for try := 0; try < maxTries; try++ {
		begin := *(*uint32)(p)
		copy(dst, src)
		end := *(*uint32)(unsafe.Add(p, 4))
		if begin == end {
			return out, true
		}
	}
	return out, false
}
