package lmu

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

// usageSample is a trimmed copy of a real /rest/strategy/usage response, with a
// couple of drivers and the player ("Steeven Rollins") carrying fuel + tyres.
const usageSample = `{
  "Conrad Laursen":[{"lap":0,"pit":false,"stint":1,"ve":1.0},{"lap":1,"pit":true,"stint":1,"ve":0.9686275124549866}],
  "Gianmaria Bruni":[{"lap":0,"pit":false,"stint":1,"ve":0.19},{"lap":1,"pit":true,"stint":1,"ve":0.16078431904315948},{"lap":2,"pit":false,"stint":1,"ve":0.12941177189350128}],
  "Steeven Rollins":[{"fuel":0.6833333373069763,"lap":0,"pit":false,"stint":1,"tyres":[97.25,98.03,93.72,95.68],"ve":1.0},{"fuel":0.6627451181411743,"lap":1,"pit":true,"stint":1,"tyres":[97.25,98.03,93.72,95.68],"ve":0.9725490808486938}]
}`

func TestFetchUsageVE(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usagePath {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(usageSample))
	}))
	defer ts.Close()

	c := newRESTClient(ts.URL)
	got, err := c.fetchUsageVE()
	if err != nil {
		t.Fatalf("fetchUsageVE: %v", err)
	}

	// Latest ve per driver (the last entry's ve).
	want := map[string]float64{
		"Conrad Laursen":  0.9686275124549866,
		"Gianmaria Bruni": 0.12941177189350128,
		"Steeven Rollins": 0.9725490808486938,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d drivers, want %d (%v)", len(got), len(want), got)
	}
	for name, w := range want {
		if math.Abs(got[name]-w) > 1e-9 {
			t.Errorf("ve[%q] = %v, want %v", name, got[name], w)
		}
	}
}

// TestVirtualEnergyLookup checks the cached lookup, name trimming, and the
// "unavailable" path.
func TestVirtualEnergyLookup(t *testing.T) {
	c := newRESTClient("http://example.invalid")

	// Before any successful poll, nothing is available.
	if _, ok := c.virtualEnergy("Steeven Rollins"); ok {
		t.Error("expected unavailable before first successful poll")
	}

	// Simulate a successful poll result.
	c.mu.Lock()
	c.veByDriver = map[string]float64{"Steeven Rollins": 0.5}
	c.ok = true
	c.mu.Unlock()

	if ve, ok := c.virtualEnergy("  Steeven Rollins  "); !ok || ve != 0.5 {
		t.Errorf("lookup with surrounding spaces = (%v,%v), want (0.5,true)", ve, ok)
	}
	if _, ok := c.virtualEnergy("Nobody"); ok {
		t.Error("unknown driver should not be found")
	}
}

// TestFetchUsageVEHTTPError verifies a non-200 response is an error (so the
// adapter falls back to shared-memory-only rather than showing bad data).
func TestFetchUsageVEHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	c := newRESTClient(ts.URL)
	if _, err := c.fetchUsageVE(); err == nil {
		t.Error("expected an error on HTTP 503")
	}
}
