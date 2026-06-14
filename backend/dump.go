// dump.go provides a compact console view of the live telemetry, enabled with
// the -dump flag. It exists for validating the LMU adapter against the running
// game: glance at these values and compare them to the in-game HUD without
// needing a browser. It prints once per second (independent of the broadcast
// rate).
package main

import (
	"fmt"
	"strings"

	"github.com/stevirn/PitMate/telemetry"
)

// dumpFrame prints a short, human-readable summary of one frame.
func dumpFrame(f telemetry.Frame) {
	if !f.Source.Connected {
		fmt.Printf("[dump] no live data — source=%q connected=false (is LMU + the shared-memory plugin running?)\n", f.Source.Game)
		return
	}

	p := f.Player
	s := f.Session
	t := p.Timing
	e := p.Energy
	sp := p.Speed
	r := p.Race
	tr := p.Tires
	sy := p.Systems

	var b strings.Builder
	b.WriteString("\n── PitMate dump ──\n")
	fmt.Fprintf(&b, "  game=%s session=%s track=%q flag=%s sc=%s timeLeft=%s\n",
		f.Source.Game, s.Type, s.TrackName, f.RaceControl.CurrentFlag, f.RaceControl.SafetyCar, fmtClock(s.RemainingSeconds))
	fmt.Fprintf(&b, "  pos P%d (class P%d) lap=%d last=%s best=%s gapAhead=%s gapBehind=%s\n",
		r.PositionOverall, r.PositionInClass, t.LapsCompleted, fmtLap(t.LastLapSeconds), fmtLap(t.BestLapSeconds), fmtGap(r.GapAheadSeconds), fmtGap(r.GapBehindSeconds))
	fmt.Fprintf(&b, "  speed=%.0f km/h gear=%d rpm=%.0f fuel=%.1fL (%.1f laps) bias=%.1f%%F oil=%.0f water=%.0f\n",
		sp.CurrentKph, sp.Gear, sp.RPM, e.FuelLitres, e.FuelLapsRemaining, sy.BrakeBiasFrontPct, sy.OilTempC, sy.WaterTempC)
	fmt.Fprintf(&b, "  tires°C FL/FR/RL/RR=%.0f/%.0f/%.0f/%.0f wear%%=%.0f/%.0f/%.0f/%.0f compound=%q\n",
		tr.FrontLeft.TempC, tr.FrontRight.TempC, tr.RearLeft.TempC, tr.RearRight.TempC,
		tr.FrontLeft.WearFraction*100, tr.FrontRight.WearFraction*100, tr.RearLeft.WearFraction*100, tr.RearRight.WearFraction*100, tr.Compound)
	fmt.Fprintf(&b, "  pit=%s stops=%d inPitLane=%t hybrid=%t/%s competitors=%d\n",
		r.PitStatus, r.PitStopCount, r.InPitLane, e.HasHybrid, e.HybridMode, len(f.Competitors))
	fmt.Print(b.String())
}

// fmtLap formats a lap/sector time as M:SS.mmm, or "—" when missing.
func fmtLap(s float64) string {
	if s <= 0 {
		return "—"
	}
	m := int(s) / 60
	sec := s - float64(m*60)
	return fmt.Sprintf("%d:%06.3f", m, sec)
}

// fmtGap formats a signed gap in seconds, or "—" when zero/missing.
func fmtGap(s float64) string {
	if s == 0 {
		return "—"
	}
	return fmt.Sprintf("%+.1f", s)
}

// fmtClock formats a duration as H:MM:SS (or M:SS under an hour), or "—".
func fmtClock(s float64) string {
	if s <= 0 {
		return "—"
	}
	sec := int(s)
	h := sec / 3600
	m := (sec % 3600) / 60
	ss := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, ss)
	}
	return fmt.Sprintf("%d:%02d", m, ss)
}
