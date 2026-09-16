package app

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwulff/steno/internal/daemon"
)

func TestLanguagePickerOpensWithBothLanguages(t *testing.T) {
	m := New()
	m.width, m.height = 100, 30
	m.connected = true
	m.client = &daemon.Client{}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	view := updated.(Model).View()
	for _, want := range []string{"Transcription language", "English", "한국어", "Enter", "Esc"} {
		if !strings.Contains(view, want) {
			t.Errorf("language picker missing %q", want)
		}
	}
}

func pickerModel() Model {
	m := New()
	m.connected, m.recording = true, true
	m.client = &daemon.Client{}
	m.engineStatus = StatusRecording
	m.locale = "en-US"
	m.width, m.height = 100, 30
	return m
}

func pickerKey(m Model, key tea.KeyMsg) (Model, tea.Cmd) {
	next, cmd := m.Update(key)
	return next.(Model), cmd
}

func TestLanguagePickerCancelDoesNotChangeLanguage(t *testing.T) {
	m := pickerModel()
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := pickerKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || m.showLanguagePicker || m.locale != "en-US" || m.pendingLanguage != "" {
		t.Fatal("cancel changed language or sent a command")
	}
}

func TestLanguagePickerWaitsForDaemonConfirmation(t *testing.T) {
	m := pickerModel()
	m.sessionID = "english-session"
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := pickerKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || m.locale != "en-US" || m.pendingLanguage != "ko-KR" || m.showLanguagePicker {
		t.Fatal("must wait for daemon confirmation before showing Korean as current")
	}
	updated, _ := m.Update(LanguageResponseMsg{Response: daemon.Response{
		OK: true, Locale: "ko-KR", SessionID: "korean-session", Status: "recording", Recording: daemon.BoolPtr(true),
	}})
	got := updated.(Model)
	if got.locale != "ko-KR" || got.pendingLanguage != "" || got.sessionID != "korean-session" {
		t.Fatal("successful response did not confirm language and new session")
	}
	if len(got.entries) != 1 || !got.entries[0].IsBoundary {
		t.Fatal("missing session boundary")
	}
	if !strings.Contains(got.renderHeader(), "한국어") {
		t.Fatal("header missing active language")
	}
}

func TestLanguagePickerFailureKeepsConfirmedLanguage(t *testing.T) {
	m := pickerModel()
	m.pendingLanguage = "ko-KR"
	updated, _ := m.Update(LanguageResponseMsg{Response: daemon.Response{OK: false, Error: "model download failed"}})
	got := updated.(Model)
	if got.locale != "en-US" || got.pendingLanguage != "" || got.errorMessage != "model download failed" {
		t.Fatal("failure must preserve confirmed language and report error")
	}
}

func TestLanguagePickerCannotResumePause(t *testing.T) {
	m := pickerModel()
	m.engineStatus, m.recording = StatusPaused, false
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := pickerKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.pendingLanguage != "" || m.recording {
		t.Fatal("language selection must not resume recording")
	}
	if !strings.Contains(m.renderLanguagePicker(), "Paused.") {
		t.Fatal("missing pause guidance")
	}
}

func TestLanguagePickerCanRetrySameLanguageAfterFailure(t *testing.T) {
	m := pickerModel()
	m.recording = false
	m, _ = pickerKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	_, cmd := pickerKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("stopped recording must be able to retry")
	}
}

func TestLanguagePickerStatusRestoresSelection(t *testing.T) {
	m := pickerModel()
	updated, _ := m.Update(StatusResponseMsg{Response: daemon.Response{OK: true, Locale: "ko-KR"}})
	m, _ = pickerKey(updated.(Model), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.selectedLanguage != 1 {
		t.Fatal("must highlight daemon's current language")
	}
}

func TestLanguagePickerQuitStillExits(t *testing.T) {
	m := pickerModel()
	m.showLanguagePicker = true
	_, cmd := pickerKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit blocked by picker")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected quit")
	}
}

func TestLanguageSelectionSendsLocaleAndPreservesAudioMode(t *testing.T) {
	m := pickerModel()
	m.showLanguagePicker = true
	m.selectedLanguage = 1
	m.systemAudio = true
	data := captureCommand(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	var command daemon.Command
	if err := json.Unmarshal(data, &command); err != nil {
		t.Fatal(err)
	}
	if command.Cmd != "reconfigure" || command.Locale != "ko-KR" || command.SystemAudio == nil || !*command.SystemAudio {
		t.Fatalf("wrong language command: %+v", command)
	}
}
