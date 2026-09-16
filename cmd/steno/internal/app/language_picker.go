package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jwulff/steno/internal/daemon"
	"github.com/jwulff/steno/internal/ui"
)

var transcriptionLanguages = []struct{ locale, label string }{
	{"en-US", "English"},
	{"ko-KR", "한국어"},
}

func languageIndex(locale string) int {
	if strings.HasPrefix(strings.ToLower(locale), "ko") {
		return 1
	}
	return 0
}

func languageLabel(locale string) string {
	if locale == "" {
		return "Language: —"
	}
	for _, option := range transcriptionLanguages {
		if strings.HasPrefix(strings.ToLower(locale), option.locale[:2]) {
			return option.label
		}
	}
	return locale
}

type LanguageResponseMsg struct {
	Response daemon.Response
	Err      error
}

func languageCmd(client *daemon.Client, locale string, systemAudio bool) tea.Cmd {
	return func() tea.Msg {
		response, err := client.SendCommand(daemon.Command{
			Cmd: "reconfigure", Locale: locale, SystemAudio: daemon.BoolPtr(systemAudio),
		})
		return LanguageResponseMsg{Response: response, Err: err}
	}
}

func (m Model) handleLanguageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyEsc, KeyLanguage:
		m.showLanguagePicker = false
	case KeyUp, KeyK:
		m.selectedLanguage = max(0, m.selectedLanguage-1)
	case KeyDown, KeyJ:
		m.selectedLanguage = min(len(transcriptionLanguages)-1, m.selectedLanguage+1)
	case KeyEnter:
		// A language change must never resume a user's pause implicitly.
		if m.engineStatus == StatusPaused || !m.connected || m.client == nil {
			return m, nil
		}
		locale := transcriptionLanguages[m.selectedLanguage].locale
		m.showLanguagePicker = false
		if strings.ReplaceAll(m.locale, "_", "-") == locale && m.recording {
			return m, nil
		}
		m.pendingLanguage = locale
		return m, languageCmd(m.client, locale, m.systemAudio)
	}
	return m, nil
}

func (m Model) renderLanguagePicker() string {
	lines := []string{ui.TitleStyle.Render("Transcription language"), ""}
	for i, option := range transcriptionLanguages {
		line := "  " + option.label
		if m.locale != "" && languageIndex(m.locale) == i {
			line += " (current)"
		}
		if i == m.selectedLanguage {
			line = ui.SelectedStyle.Render("› " + strings.TrimPrefix(line, "  "))
		}
		lines = append(lines, line)
	}
	hint := "↑/↓ or j/k: move · Enter: select · Esc: cancel"
	if m.engineStatus == StatusPaused {
		hint = "Paused. Esc, then p to resume before changing."
	}
	if !m.connected {
		hint = "Waiting for the daemon… Esc: cancel"
	}
	lines = append(lines, "", hint)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ui.ColorCyan).
		Padding(1, 2).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.transcriptVisibleLines(), lipgloss.Center, lipgloss.Center, box)
}
