package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwulff/steno/internal/daemon"
)

type FastModeResponseMsg struct {
	Response daemon.Response
	Err      error
}

func fastModeCmd(client *daemon.Client, enabled, systemAudio bool) tea.Cmd {
	return func() tea.Msg {
		response, err := client.SendCommand(daemon.Command{
			Cmd: "reconfigure", SystemAudio: daemon.BoolPtr(systemAudio), LowLatencyTranscription: daemon.BoolPtr(enabled),
		})
		return FastModeResponseMsg{Response: response, Err: err}
	}
}
