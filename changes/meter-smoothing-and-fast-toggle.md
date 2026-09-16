# Calm audio meters and expose recognition mode in the TUI

The previous -60 dBFS meter exaggerated background sound and displayed every empty level-reporting window as silence. Live observation showed a repeating nonzero/nonzero/zero pattern, explaining the flashing bars. Change the display floor to -48 dBFS, round cells, use an 80 ms attack / 180 ms peak hold / 24 dB-per-second release, and add a 0.3-cell hysteresis band. Silence eventually clears the meter; pause/disconnection clears it immediately. Raw audio levels and capture gain are unchanged.

The low-latency option previously existed only in settings.json. Press `f` to switch it directly in the TUI. Show the daemon-confirmed mode in the header/footer, prevent duplicate in-flight reconfigurations, preserve language/device/system-audio selections, and save successful changes for the next launch. Failed switches restore the prior mode. While paused, a mode-only change is saved without resuming capture or resetting its deadline. Language/source changes still require a resume. A TUI connected to an older daemon requests a restart instead of sending an unsupported reconfiguration.

Pass the selected mode to every recognizer creation, including source recovery and resume. The production factory applies fastResults when requested. Keep the existing initializer and factory interface available for callers that use their own default options.

Version: 0.5.2-tui.4.


Validation: 577 daemon tests, 43 app tests, and 210 Go tests passed; five live-daemon tests were explicitly disabled. The observed repeating zero-gap pattern remains visible without flashing off, quiet speech still registers, background noise is reduced, and sustained silence clears the display. Socket-command tests verify explicit on/off values, daemon-confirmed UI state, old-daemon handling, failure rollback, and preservation of pause state. Separate isolated settings checks verified persisted on/off values and paused-mode saving. No live reconfiguration was performed against the user's running daemon.
