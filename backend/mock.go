// mock.go provides a synthetic telemetry source for developing and testing the
// pipeline without the game running. It is part of package main (the binary),
// not an adapter, because it is a test tool rather than a real game.
//
// Enable it with the -mock flag. It produces plausible, continuously moving
// values (lap progress, draining fuel, oscillating tire temps) so the broadcast
// loop and any UI can be exercised end to end.
package main

import (
	"fmt"
	"math"

	"github.com/stevirn/PitMate/telemetry"
)

// mockSource generates synthetic frames. Each call to Read advances its internal
// state by one tick, so the data appears to move over time.
type mockSource struct {
	tick int
}

// newMockSource creates a mock data source.
func newMockSource() *mockSource { return &mockSource{} }

// Read returns the next synthetic frame and advances internal state.
func (m *mockSource) Read() telemetry.Frame {
	m.tick++
	t := float64(m.tick)

	// Lap progresses 0..1 and wraps; fuel drains slowly over the session.
	lapFraction := math.Mod(t/100.0, 1.0)
	lapsDone := m.tick / 100
	fuel := math.Max(0, 80.0-0.05*t)

	// A gently oscillating value generator for temps/speeds, so the UI moves.
	osc := func(base, amp, period float64) float64 {
		return base + amp*math.Sin(t*2*math.Pi/period)
	}

	player := telemetry.Car{
		Identity: telemetry.Identity{
			CarID:      "player",
			CarNumber:  "7",
			CarClass:   "Hypercar",
			CarModel:   "Toyota GR010",
			DriverName: "Mock Driver",
			IsPlayer:   true,
		},
		Timing: telemetry.Timing{
			LapsCompleted:     lapsDone,
			LapFraction:       lapFraction,
			CurrentLapSeconds: lapFraction * 210,
			LastLapSeconds:    osc(210, 1.5, 30),
			BestLapSeconds:    208.4,
			LastSectors:       []float64{osc(62, 0.5, 23), osc(78, 0.6, 29), osc(70, 0.4, 31)},
			BestSectors:       []float64{61.6, 77.4, 69.4},
		},
		Energy: telemetry.Energy{
			FuelLitres:            fuel,
			FuelCapacityLitres:    80,
			FuelPerLastLap:        3.4,
			FuelPerLapAvg:         3.5,
			FuelLapsRemaining:     fuel / 3.5,
			HasHybrid:             true,
			HybridChargeFraction:  math.Mod(t/40.0, 1.0),
			HybridMode:            "Attack",
			HasVirtualEnergy:      true,
			VirtualEnergyFraction: math.Max(0, 1.0-0.0004*t),
		},
		Tires: telemetry.Tires{
			Compound:   "Medium",
			FrontLeft:  mockTire(osc(92, 4, 17), 0.95-0.0005*t),
			FrontRight: mockTire(osc(95, 4, 19), 0.95-0.0005*t),
			RearLeft:   mockTire(osc(98, 5, 21), 0.93-0.0006*t),
			RearRight:  mockTire(osc(100, 5, 23), 0.93-0.0006*t),
		},
		Speed: telemetry.Speed{
			CurrentKph: math.Abs(osc(180, 120, 18)),
			TopKph:     330,
			Gear:       4,
			RPM:        osc(7000, 1500, 9),
		},
		Systems: telemetry.Systems{
			BrakeBiasFrontPct: 55.5,
			TractionControl:   3,
			ABS:               2,
			OilTempC:          osc(105, 3, 60),
			WaterTempC:        osc(88, 2, 70),
		},
		Race: telemetry.RaceState{
			PositionOverall:    2,
			PositionInClass:    2,
			GapToLeaderSeconds: osc(4.5, 1.5, 40),
			GapAheadSeconds:    osc(2.1, 0.8, 25),
			GapBehindSeconds:   osc(1.4, 0.6, 27),
			PitStatus:          telemetry.PitNone,
		},
	}

	competitors := []telemetry.Car{
		mockCompetitor("1", "Ferrari 499P", 1, -osc(2.1, 0.8, 25), lapsDone),
		mockCompetitor("51", "Porsche 963", 3, osc(1.4, 0.6, 27), lapsDone),
	}

	return telemetry.Frame{
		// Timestamp and Sequence are stamped by the server on Broadcast.
		Source: telemetry.SourceInfo{
			Game:      "Mock",
			AdapterID: "mock",
			Connected: true,
			Version:   "dev",
		},
		Session: telemetry.SessionInfo{
			Type:             telemetry.SessionRace,
			TrackName:        "Le Mans 24h",
			TrackLengthM:     13626,
			ElapsedSeconds:   t,
			RemainingSeconds: math.Max(0, 86400-t),
			IsTimed:          true,
			Conditions: telemetry.Conditions{
				AirTempC:      osc(22, 2, 300),
				TrackTempC:    osc(31, 3, 300),
				TrackWetness:  0,
				RainIntensity: 0,
			},
		},
		Player:      player,
		Competitors: competitors,
		RaceControl: telemetry.RaceControl{
			CurrentFlag: telemetry.FlagGreen,
			SafetyCar:   telemetry.SafetyCarNone,
		},
		Events: []telemetry.Event{
			{Type: telemetry.EventInfo, Message: "Mock session running", Severity: 0},
		},
	}
}

// mockTire builds one tire corner from a temperature and a wear fraction.
func mockTire(tempC, wear float64) telemetry.TireCorner {
	return telemetry.TireCorner{
		TempC:        tempC,
		PressureKpa:  165,
		WearFraction: math.Max(0, wear),
		BrakeTempC:   tempC + 250,
	}
}

// mockCompetitor builds a minimal competitor car (the fields visible from
// outside): identity, position, and gap.
func mockCompetitor(number, model string, pos int, gap float64, lapsDone int) telemetry.Car {
	return telemetry.Car{
		Identity: telemetry.Identity{
			CarID:      fmt.Sprintf("car-%s", number),
			CarNumber:  number,
			CarClass:   "Hypercar",
			CarModel:   model,
			DriverName: "Rival " + number,
		},
		Timing: telemetry.Timing{LapsCompleted: lapsDone},
		Race: telemetry.RaceState{
			PositionOverall: pos,
			PositionInClass: pos,
			GapAheadSeconds: gap,
			PitStatus:       telemetry.PitNone,
		},
	}
}
