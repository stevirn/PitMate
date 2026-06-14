package main

import (
	"encoding/json"
	"testing"
)

// TestMockSourceProducesMovingFrames verifies the mock source advances state
// each Read (so the UI sees movement) and produces JSON-encodable frames.
func TestMockSourceProducesMovingFrames(t *testing.T) {
	m := newMockSource()

	f1 := m.Read()
	f2 := m.Read()

	if f1.Session.ElapsedSeconds == f2.Session.ElapsedSeconds {
		t.Error("expected elapsed time to advance between reads")
	}
	if f1.Player.Identity.CarNumber != "7" {
		t.Errorf("player car number = %q, want \"7\"", f1.Player.Identity.CarNumber)
	}
	if len(f1.Competitors) != 2 {
		t.Errorf("competitors = %d, want 2", len(f1.Competitors))
	}
	if !f1.Source.Connected {
		t.Error("mock source should report Connected=true")
	}

	if _, err := json.Marshal(f2); err != nil {
		t.Fatalf("frame failed to JSON-encode: %v", err)
	}
}
