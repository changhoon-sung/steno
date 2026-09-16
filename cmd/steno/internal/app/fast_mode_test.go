package app

import (
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwulff/steno/internal/daemon"
	"strings"
	"testing"
)

func TestFastModeKeySendsToggle(t *testing.T) {
	m := New()
	m.connected = true
	m.lowLatencySupported = true
	data := captureCommand(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	var command map[string]any
	if err := json.Unmarshal(data, &command); err != nil {
		t.Fatal(err)
	}
	if command["cmd"] != "reconfigure" || command["lowLatencyTranscription"] != true {
		t.Fatalf("f did not request low-latency mode: %s", data)
	}
}

func TestFastModeOffIsExplicitOnWire(t *testing.T) {
	m := New()
	m.connected = true
	m.lowLatencySupported = true
	m.lowLatencyTranscription = true
	data := captureCommand(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	var command map[string]any
	if err := json.Unmarshal(data, &command); err != nil {
		t.Fatal(err)
	}
	if mode, ok := command["lowLatencyTranscription"]; !ok || mode != false {
		t.Fatalf("off was omitted: %s", data)
	}
}

func TestFastModeWaitsForDaemonAndShowsState(t *testing.T) {
	m := New()
	m.connected = true
	m.lowLatencySupported = true
	m.client = &daemon.Client{}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(Model)
	if cmd == nil || !m.modePending || m.lowLatencyTranscription {
		t.Fatal("mode changed before confirmation")
	}
	updated, _ = m.Update(FastModeResponseMsg{Response: daemon.Response{OK: true, LowLatencyTranscription: daemon.BoolPtr(true)}})
	m = updated.(Model)
	if !m.lowLatencyTranscription || m.modePending {
		t.Fatal("daemon confirmation not applied")
	}
	if !strings.Contains(m.renderHeader(), "LOW LATENCY") || !strings.Contains(m.renderFooter(), "Low latency:on") {
		t.Fatal("mode must be visible")
	}
}

func TestFastModeDoesNotResumePausedRecording(t *testing.T) {
	m := New()
	m.connected = true
	m.lowLatencySupported = true
	m.client = &daemon.Client{}
	m.engineStatus = StatusPaused
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(Model)
	if cmd == nil || !m.modePending || m.recording {
		t.Fatal("paused mode should request a preference change only")
	}
	updated, _ = m.Update(FastModeResponseMsg{Response: daemon.Response{
		OK: true, LowLatencyTranscription: daemon.BoolPtr(true), Recording: daemon.BoolPtr(false),
		Status: "paused", Paused: daemon.BoolPtr(true), PausedIndefinitely: daemon.BoolPtr(true),
	}})
	m = updated.(Model)
	if m.recording || m.engineStatus != StatusPaused || !m.pausedIndefinitely || !m.lowLatencyTranscription {
		t.Fatal("mode response changed pause state")
	}
}

func TestFastModeDoesNotReconfigureOldDaemon(t *testing.T) {
	m := New()
	m.connected = true
	m.client = &daemon.Client{}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if updated.(Model).modePending || !strings.Contains(updated.(Model).errorMessage, "Restart") {
		t.Fatal("old daemon must not be restarted by an unsupported toggle")
	}
}

func TestFastModeRestoresDisabledState(t *testing.T) {
	m := New()
	m.lowLatencyTranscription = true
	updated, _ := m.Update(StatusResponseMsg{Response: daemon.Response{OK: true, LowLatencyTranscription: daemon.BoolPtr(false)}})
	m = updated.(Model)
	if m.lowLatencyTranscription || !m.lowLatencySupported {
		t.Fatal("explicit false status must restore accurate mode")
	}
}
