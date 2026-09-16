# Steno

A fast, private speech-to-text TUI for macOS.

Steno uses Apple's SpeechAnalyzer API (macOS 26) for real-time transcription that runs entirely on-device. No cloud services, no API keys, no rate limits.

This fork provides a transcription-only TUI. Local summaries, meeting notes, and topic generation are disabled; previously saved data remains readable through MCP.

## Requirements

- macOS 26 (Tahoe) or later
- Apple Silicon (arm64)
- Microphone access

## Install

### This fork

This fork defaults to TUI-owned daemon lifetime. Build it from source:

```bash
git clone https://github.com/changhoon-sung/steno.git
cd steno
git checkout feat/macos27-transcription
make install
```

Install location: `~/.local/bin`. Upstream release downloads below do not include this change.

### Upstream download

Download the latest release from [GitHub Releases](https://github.com/jwulff/steno/releases/latest):

```bash
# Download and extract
curl -LO https://github.com/jwulff/steno/releases/latest/download/steno-darwin-arm64.tar.gz
tar xzf steno-darwin-arm64.tar.gz

# Install to ~/.local/bin (make sure it's in your PATH)
mkdir -p ~/.local/bin
mv steno steno-daemon ~/.local/bin/
```

### From source

Requires Xcode 27 (Swift 6.4 / macOS 27 SDK) and Go 1.24+. macOS 26 retains the compatible audio-conversion path.

```bash
git clone https://github.com/jwulff/steno.git
cd steno
make install   # Builds, signs, and installs to ~/.local/bin
```

## Usage

```bash
steno            # Launch TUI — auto-starts the daemon
steno --mcp      # Run as MCP stdio server (for Claude Desktop, etc.)
```

Running `steno` starts a daemon owned by that TUI. Quitting with `q`, closing the terminal, or killing the TUI also stops its daemon and audio capture. Shutdown normally drains pending transcripts; a five-second deadline forces exit if draining gets stuck. The owner is monitored before audio initialization, so exiting during startup stops it too.

`steno --mcp` only reads saved transcripts. It does not start a daemon or record audio and remains useful with the TUI closed. If several TUIs share one daemon, closing its original owner stops that daemon; another open TUI reconnects and starts a replacement owned by itself.

### Controls

| Key | Action |
|-----|--------|
| `Space` | Start a new session |
| `p` / `P` | Pause for 30 minutes / indefinitely; press again to resume |
| `i` | Cycle input devices |
| `a` | Toggle system audio capture |
| `j`/`k` | Scroll transcript |
| `Up`/`Down` | Scroll transcript |
| `l` | Select transcription language (English / 한국어) |
| `q` | Quit and stop the daemon owned by this TUI |

### Audio meters and low-latency mode

MIC shows microphone input; SYS shows audio played by the computer. The meters use a -60 to 0 dBFS display scale so ordinary quiet speech remains visible. This changes the display, not recording gain.

On macOS 27, audio is converted with `AnalyzerInputConverter`, including its final buffered tail. The microphone uses the new read-only audio tap and transfers an owned copy to the asynchronous pipeline.

The optional `lowLatencyTranscription` setting in `settings.json` enables SpeechTranscriber's `fastResults`. It defaults to `false`: a short English/Korean streaming comparison improved partial-result latency but did not materially accelerate final results, and Korean character errors increased. Change it only while Steno is stopped; the next launch reads it.

### Transcription language

Press `l`, use `↑` / `↓` (or `j` / `k`) to choose **English** or **한국어**, and press `Enter`. `Esc` cancels. The header shows the language confirmed by the daemon; the choice is saved for the next launch. Changing language briefly restarts recording in a new session. On first use, Apple's speech model may need to download.

While paused, the picker shows a reminder to close it with `Esc` and resume with `p` before changing language. Selecting a language never implicitly resumes a pause.

### MCP Server

Steno includes a built-in [MCP](https://modelcontextprotocol.io) server for querying your transcript database from AI tools like Claude Desktop.

Add to your MCP client config:

```json
{
  "mcpServers": {
    "steno": {
      "command": "steno",
      "args": ["--mcp"]
    }
  }
}
```

Available tools: `get_overview`, `list_sessions`, `get_session`, `get_transcript`, `search`.

### Daemon Management

For TUI-only recording, launch `steno` and do not install a background service. Stop any previously installed Homebrew/launchd service and standalone daemon before switching to this mode: an already-running independent daemon has no TUI owner.

Advanced: explicitly launching `steno-daemon run` without `--owner-pid` or installing a launchd service still enables independent background recording:

```bash
steno-daemon run         # Run daemon in foreground
steno-daemon status      # Check if daemon is running
steno-daemon install     # Install as launchd service (auto-start on login)
steno-daemon uninstall   # Remove launchd service
```

## How It Works

Steno uses the SpeechAnalyzer API introduced in macOS 26, which provides:

- **On-device processing** — your audio never leaves your Mac
- **Low latency** — real-time transcription as you speak
- **High accuracy** — 55% faster than Whisper Large V3 Turbo in Apple's benchmarks

## Architecture

Steno is a two-process system: a Swift daemon handles audio capture and speech recognition, while a Go binary provides the TUI and MCP server.

```
┌─────────────────┐         Unix socket          ┌──────────────────────┐
│   steno         │◄──── NDJSON commands ────────►│   steno-daemon       │
│   (Go)          │◄──── NDJSON events ──────────►│   (Swift)            │
│                 │                               │                      │
│  - TUI display  │                               │  - Microphone capture│
│  - MCP server   │      SQLite (read-only)       │  - System audio      │
│  - Daemon mgmt  │◄─────────────────────────────►│  - SpeechAnalyzer    │
│  - Level meters │                               │  - Segment storage   │
└─────────────────┘                               │  - Segment storage   │
                                                  └──────────────────────┘
```

- **`steno`** (Go) — TUI + MCP server + daemon lifecycle management. Connects to the daemon via Unix socket and provides saved-transcript MCP queries.
- **`steno-daemon`** (Swift) — Captures mic + system audio via ScreenCaptureKit, runs SpeechAnalyzer/SpeechTranscriber, persists segments to SQLite (GRDB). The production daemon does not instantiate a summary coordinator or local LLM.

## Project Structure

```
steno/
├── daemon/                    # Swift daemon (steno-daemon)
│   ├── Package.swift
│   ├── Sources/StenoDaemon/
│   │   ├── Audio/             # Mic + system audio capture
│   │   ├── Commands/          # CLI subcommands (run, status, install)
│   │   ├── Dispatch/          # Command dispatcher, event broadcaster
│   │   ├── Engine/            # Recording engine, speech recognizer
│   │   ├── Infrastructure/    # Paths, PID file, signal handling
│   │   ├── Models/            # Domain models
│   │   ├── Permissions/       # TCC permission checks
│   │   ├── Services/          # Summarization, topic extraction
│   │   ├── Socket/            # Unix socket server, NDJSON protocol
│   │   └── Storage/           # SQLite via GRDB
│   └── Tests/StenoDaemonTests/
├── cmd/steno/                 # Go binary (steno)
│   ├── go.mod
│   ├── main.go                # Entry point: --mcp flag dispatches mode
│   └── internal/
│       ├── app/               # Bubbletea TUI model, messages, keybindings
│       ├── daemon/            # Socket client, protocol types, lifecycle manager
│       ├── db/                # SQLite read-only queries (shared by TUI + MCP)
│       ├── mcp/               # MCP tool handlers
│       └── ui/                # Lipgloss styles
└── schema/                    # SQLite schema contract
```

## Development

```bash
make build          # Build daemon (release) + steno
make test           # Run all test suites (daemon + steno)
make test-daemon    # Daemon tests only (Swift)
make test-steno     # Steno tests only (Go)
make run-daemon     # Build, sign, and run daemon (debug)
make run-steno      # Build and run TUI
make run-mcp        # Build and run MCP server
make clean          # Remove all build artifacts
make install        # Install to ~/.local/bin (override with PREFIX=)
```

See [CLAUDE.md](CLAUDE.md) for development conventions.

## License

MIT

### Test isolation

Normal Go tests skip the live-daemon tests. `STENO_LIVE_TESTS=1` explicitly enables tests that can stop and restart the default daemon; use it only with a disposable recording session. Set `STENO_SETTINGS_PATH` to a temporary file when running Swift tests to keep personal settings separate.
