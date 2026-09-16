# Select transcription language in the TUI

Press `l` to open a centered English / 한국어 picker. Arrow keys or j/k move, Enter selects, and Esc cancels. The header displays the daemon-confirmed language and shows a pending change while the speech model is prepared.

The picker uses the existing reconfigure command with an explicit locale and the current system-audio setting. A successful switch starts a new session, clears partial text, and adds a visible boundary. The daemon reports its current locale, persists the last successful choice, and restores it at startup. Failures remain visible and can be retried; returning to a previously prepared language clears a failed model's stale unavailable status.

Both the TUI and daemon refuse to reconfigure paused recording. Selecting a language therefore cannot resume an indefinite pause. `q`, Ctrl-C, terminal closure, and TUI crash retain the existing owner-bound daemon shutdown behavior.

Version: 0.5.2-tui.2. No new dependencies. `STENO_SETTINGS_PATH` isolates test/development settings from personal configuration, and settings writes are atomic.

Validation on macOS 27 / Xcode 27:

- 571 daemon tests, 43 app tests, and 198 Go tests passed; 5 upstream opt-in live tests skipped.
- Signed release build and signature verification passed. Existing upstream compiler warnings remain.
- Real TUI key input verified picker cancellation, English → Korean, Korean → English, persisted selection across a full TUI/daemon restart, and indefinite-pause protection.
- Actual Korean test speech was transcribed successfully. Temporary system-audio capture was returned to its original setting afterwards.
- Quitting the TUI still shut down its owned daemon.
