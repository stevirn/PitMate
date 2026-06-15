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
	"log"
	"time"
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
	size   uintptr // actual mapped region size (page-rounded), from VirtualQuery
	want   uintptr // size of the Go struct we expect to read (for diagnostics)
}

// winReader holds the two mapped buffers PitMate reads. They are opened lazily
// on the first successful read so the adapter can start before the game does.
type winReader struct {
	tele      *mappedBuffer
	score     *mappedBuffer
	connected bool      // were both buffers readable on the previous attempt?
	lastLog   time.Time // rate-limits the diagnostic logging
}

// newReader returns a Windows reader. Opening the shared memory is deferred to
// read() so a not-yet-running game is not treated as a fatal error.
func newReader() (reader, error) { return &winReader{}, nil }

// note logs a diagnostic at most once every few seconds, so a persistent
// failure at the read rate (e.g. 10x/sec) does not flood the console.
func (r *winReader) note(format string, args ...any) {
	now := time.Now()
	if now.Sub(r.lastLog) < 3*time.Second {
		return
	}
	r.lastLog = now
	log.Printf("lmu: "+format, args...)
}

// openMapping opens an existing named mapping created by the plugin and maps a
// read-only view of it. The plugin creates the mapping in the Local namespace,
// so this must run in the same Windows session as the game (it does — same PC).
//
// We pass length 0 to MapViewOfFile so it maps the ENTIRE object regardless of
// size — passing an explicit size that exceeds the object fails with
// ACCESS_DENIED. We then ask VirtualQuery for the real region size and bound all
// reads to it, so a layout mismatch is diagnosable rather than a crash.
func openMapping(name string, want uintptr) (*mappedBuffer, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := openFileMapping(windows.FILE_MAP_READ, false, namePtr)
	if err != nil {
		return nil, fmt.Errorf("open mapping %s: %w", name, err)
	}
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("map view %s: %w", name, err)
	}

	var mbi windows.MemoryBasicInformation
	region := uintptr(0)
	if err := windows.VirtualQuery(addr, &mbi, unsafe.Sizeof(mbi)); err == nil {
		region = mbi.RegionSize
	}
	return &mappedBuffer{handle: h, base: addr, size: region, want: want}, nil
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
// snapshots of both buffers. ok is false until both are open and readable. It
// logs (rate-limited) why it can't read, which is the main tool for diagnosing
// a "no data" situation: the Windows error distinguishes "mapping not found"
// (plugin not loaded / no session) from "access denied" (a size/layout problem).
func (r *winReader) read() (tel rf2Telemetry, sc rf2Scoring, ok bool) {
	if r.tele == nil {
		b, err := openMapping(mmTelemetryName, unsafe.Sizeof(tel))
		if err != nil {
			r.connected = false
			r.note("waiting for telemetry shared memory: %v — is LMU running with the plugin enabled and a session loaded?", err)
			return tel, sc, false
		}
		r.tele = b
	}
	if r.score == nil {
		b, err := openMapping(mmScoringName, unsafe.Sizeof(sc))
		if err != nil {
			r.connected = false
			r.note("waiting for scoring shared memory: %v", err)
			return tel, sc, false
		}
		r.score = b
	}

	t, okT := readVersioned[rf2Telemetry](r.tele.base, r.tele.size)
	s, okS := readVersioned[rf2Scoring](r.score.base, r.score.size)
	if !okT || !okS {
		r.note("shared memory opened but snapshots are inconsistent (telemetry ok=%v, scoring ok=%v)", okT, okS)
		return tel, sc, false
	}

	if !r.connected {
		r.connected = true
		// Log the buffer sizes once. If the plugin's buffer is smaller than the
		// struct we expect, our layout is wrong — this surfaces it immediately.
		log.Printf("lmu: connected to shared memory — telemetry buffer=%d bytes (struct %d), scoring buffer=%d bytes (struct %d); %d vehicles",
			r.tele.size, r.tele.want, r.score.size, r.score.want, t.mNumVehicles)
		if r.tele.size < r.tele.want || r.score.size < r.score.want {
			log.Printf("lmu: WARNING — a mapped buffer is smaller than expected; telemetry values may be wrong (layout/version mismatch)")
		}
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
// reading them at offsets 0 and 4. region is the actual mapped size; the copy is
// bounded by it so a too-small buffer can never read past the mapping.
func readVersioned[T any](base, region uintptr) (T, bool) {
	var out T
	size := unsafe.Sizeof(out)
	n := size
	if region != 0 && region < n {
		n = region
	}

	// The one unavoidable conversion: base is a valid address returned by
	// MapViewOfFile and stays valid until UnmapViewOfFile. go vet's unsafeptr
	// analyzer cannot know that and flags it; it is correct here. This is the
	// only place in the package that does it.
	p := unsafe.Pointer(base) //nolint:govet // syscall-returned mapped address
	dst := unsafe.Slice((*byte)(unsafe.Pointer(&out)), size)[:n]
	src := unsafe.Slice((*byte)(p), n)

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
