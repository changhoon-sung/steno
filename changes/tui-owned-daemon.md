# Tie automatically started daemons to the TUI

Previously a daemon spawned by the TUI survived its owner and continued recording after `q`, terminal closure, or a TUI crash. The TUI now passes `--owner-pid` when spawning it, and the daemon uses a macOS process-exit source to detect owner death, including SIGKILL.

Monitoring begins before permissions and audio initialization. Owner death requests normal SIGTERM shutdown, with a five-second hard deadline if draining hangs. The TUI reaps exited child processes while it remains alive. MCP readers continue to access saved transcripts without starting audio capture.

Explicit standalone `steno-daemon run` and launchd service behavior remain available. Stop old independent daemons/services before using TUI-owned recording. When multiple TUIs share a daemon, closing the original owner stops that daemon; another open TUI reconnects and owns its replacement.

Validation: subprocess tests for owner SIGTERM/SIGKILL, a live owner, invalid/dead owners, and the TUI's actual daemon arguments. Integration validation also exercises normal TUI quit, terminal closure, forced TUI death, exit during startup, and MCP access with the daemon stopped.

The fork reports version `0.5.2-tui.1`. `make install` also copies FluidAudio's resource bundle alongside the CLI binaries, as required by the Swift 6.4 resource accessor.

Validated locally on macOS 27 with Xcode 27 / Swift 6.4:

- 566 daemon tests, 43 app tests, and 189 Go tests passed; 5 upstream opt-in live tests were skipped.
- Signed release build passed signature verification. Existing upstream deprecation/test warnings remain.
- Real recording stopped with the daemon on TUI `q`, SIGTERM, SIGKILL, and terminal closure (0.04–0.27 seconds observed). Killing the TUI during daemon startup also stopped the daemon.
