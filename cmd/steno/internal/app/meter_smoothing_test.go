package app

import (
	"strings"
	"testing"
	"time"
)

func TestMeterDoesNotExaggerateBackgroundNoise(t *testing.T) {
	for _, peak := range []float32{0.001, 0.003, 0.005} {
		if got := strings.Count(renderLevelMeter("MIC", peak), "█"); got != 0 {
			t.Errorf("background peak %v fills %d cells", peak, got)
		}
	}
	if got := strings.Count(renderLevelMeter("MIC", 0.08), "█"); got > 4 {
		t.Errorf("ordinary speech fills %d cells, want at most 4", got)
	}
}

func TestMeterBridgesObservedZeroGaps(t *testing.T) {
	var meter meterEnvelope
	at := time.Unix(100, 0)
	previous, changes := -1, 0
	// The daemon alternates real peaks with empty 100 ms windows.
	for i := 0; i < 60; i++ {
		peak := float32(0.08)
		if i%3 == 2 {
			peak = 0
		}
		now := at.Add(time.Duration(i) * 100 * time.Millisecond)
		meter.update(peak, now)
		cells := meter.cellsAt(now)
		if cells == 0 {
			t.Fatalf("short empty window blanked the meter at tick %d", i)
		}
		if previous >= 0 && cells != previous {
			changes++
		}
		previous = cells
	}
	if changes > 2 {
		t.Fatalf("meter still flickers: %d changes", changes)
	}
}

func TestMeterEventuallyFallsToSilence(t *testing.T) {
	var meter meterEnvelope
	at := time.Unix(100, 0)
	meter.update(0.08, at)
	if got := meter.cellsAt(at.Add(100 * time.Millisecond)); got == 0 {
		t.Fatal("short gap must retain the level")
	}
	if got := meter.cellsAt(at.Add(2 * time.Second)); got != 0 {
		t.Fatalf("meter stayed lit after silence: %d", got)
	}
}

func TestMeterHysteresisPreventsBoundaryChatter(t *testing.T) {
	var meter meterEnvelope
	at := time.Unix(100, 0)
	meter.update(0.044, at)
	previous := meter.cellsAt(at)
	changes := 0
	for i := 1; i < 40; i++ {
		peak := float32(0.044)
		if i%2 == 0 {
			peak = 0.045
		}
		now := at.Add(time.Duration(i) * 100 * time.Millisecond)
		meter.update(peak, now)
		cells := meter.cellsAt(now)
		if cells != previous {
			changes++
		}
		previous = cells
	}
	if changes > 1 {
		t.Fatalf("adjacent cells chatter: %d changes", changes)
	}
}
