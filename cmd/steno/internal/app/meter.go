package app

import (
	"math"
	"time"
)

const meterCells = 8

// A quieter display floor and rounded cells avoid amplifying background hiss.
func meterFraction(peak float32) float64 {
	p := float64(peak)
	if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
		return 0
	}
	db := 20 * math.Log10(math.Min(p, 1))
	return math.Max(0, (db+48)/48)
}

// Display-only envelope: fast attack, a short peak hold to bridge empty
// capture windows, then a 24 dB/s fall. Raw levels and recording gain stay intact.
type meterEnvelope struct {
	value     float64
	cells     int
	updatedAt time.Time
	holdUntil time.Time
}

func (m *meterEnvelope) update(peak float32, now time.Time) {
	target := meterFraction(peak)
	if m.updatedAt.IsZero() {
		m.value = target
		m.holdUntil = now.Add(180 * time.Millisecond)
	} else {
		dt := math.Max(0, now.Sub(m.updatedAt).Seconds())
		if target >= m.value {
			m.value += (target - m.value) * (1 - math.Exp(-dt/0.08))
			m.holdUntil = now.Add(180 * time.Millisecond)
		} else if now.After(m.holdUntil) {
			from := m.updatedAt
			if m.holdUntil.After(from) {
				from = m.holdUntil
			}
			m.value = math.Max(target, m.value-0.5*now.Sub(from).Seconds())
		}
	}
	m.updatedAt = now
	position := math.Max(0, math.Min(1, m.value)) * meterCells
	next := int(math.Round(position))
	// Keep adjacent cells stable near a threshold (0.3-cell dead band).
	if next > m.cells && position < float64(m.cells)+0.65 {
		return
	}
	if next < m.cells && position > float64(m.cells)-0.65 {
		return
	}
	m.cells = next
}

// A render can age the display without mutating model state. This also lets
// the existing status tick fade a stale input if level events stop arriving.
func (m meterEnvelope) cellsAt(now time.Time) int {
	if m.updatedAt.IsZero() {
		return 0
	}
	m.update(0, now)
	return m.cells
}

func (m *Model) resetLevelMeters() {
	m.micLevel, m.sysLevel = 0, 0
	m.micMeter, m.sysMeter = meterEnvelope{}, meterEnvelope{}
}

func renderSmoothedMeter(label string, raw float32, meter meterEnvelope, now time.Time) string {
	if meter.updatedAt.IsZero() {
		return renderLevelMeter(label, raw)
	}
	return renderMeterCells(label, meter.cellsAt(now))
}
