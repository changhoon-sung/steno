package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"math"
	"strings"
	"testing"
)

func TestQuietSpeechLightsLevelMeter(t *testing.T) {
	// Observed live microphone peaks: 0.0145–0.0405. The old linear
	// eight-cell meter rounded all of these down to an empty bar.
	for _, peak := range []float32{0.0145, 0.0405} {
		bar := renderLevelMeter("MIC", peak)
		if strings.Count(bar, "█") < 2 {
			t.Errorf("quiet speech %v is invisible: %s", peak, bar)
		}
	}
}

func TestLevelMeterSilenceClippingAndInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		level  float32
		filled int
	}{
		{0, 0}, {-1, 0}, {0.0001, 0}, {float32(math.NaN()), 0}, {float32(math.Inf(1)), 0}, {1, 8}, {2, 8},
	} {
		if got := strings.Count(renderLevelMeter("SYS", tc.level), "█"); got != tc.filled {
			t.Errorf("level %v: %d cells, want %d", tc.level, got, tc.filled)
		}
	}
}

func TestTranscriptionViewDoesNotOfferSummaryOrTopics(t *testing.T) {
	m := New()
	m.width, m.height = 100, 30
	m.connected = true
	view := m.View()
	for _, unwanted := range []string{"TOPICS", "No topics yet", " Summary", " Focus"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("unneeded feature still shown: %q", unwanted)
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if updated.(Model).showSummary || cmd != nil {
		t.Fatal("s must not start summary work")
	}
}
