# macOS 27 transcription path and visible audio meters

The TUI used a linear eight-cell display, so every peak below 0.125 appeared silent even when speech recognition worked. Use a -60...0 dBFS display scale, with silence/invalid-value handling and no change to capture gain or the raw level protocol. MIC and SYS represent separate capture sources.

The live daemon no longer constructs a summary coordinator or Foundation Models summarizer. It only captures and transcribes; the TUI uses its full width for the transcript. Old stored summaries and MCP query shapes remain compatible. An optional coordinator remains available to existing library tests, but the production startup path does not supply one. Speaker labeling is unchanged.

On macOS 27, AnalyzerAudioFeeder uses Apple's AnalyzerInputConverter and flushes held-over audio before closing input. The microphone's read-only audio tap hands an owned PCM copy to the pipeline, as conversion may retain buffers. A macOS 26 runtime fallback handles single-use converter input and drains its tail. Build with Xcode 27.

The optional lowLatencyTranscription setting defaults off. It enables fastResults without changing the language picker or shutdown behavior.

A test-safety issue was found during verification: the existing Go live tests auto-attached whenever the user's daemon existed. All five now require STENO_LIVE_TESTS=1; normal runs skip them even when a daemon is running. Swift test settings are isolated with STENO_SETTINGS_PATH.

## Short synthetic streaming comparison

The same generated audio files were fed in real time through the actual new recognizer pipeline on macOS 27. These are single trials, not a general accuracy benchmark. Text error excludes punctuation/case (English word error; Korean character error with spaces removed).

| Language / mode | First result | First final | Text error |
|---|---:|---:|---:|
| English / standard | 4.25 s | 8.23 s | 0% |
| English / fast | 1.13 s | 8.25 s | 0% |
| Korean / standard | 12.13 s | 14.86 s | 1.33% |
| Korean / fast | 1.10 s | 14.73 s | 4.00% |

Fast mode helps the visible partial transcript, but this test did not show a useful improvement in final-result availability for SQLite/MCP. Preserve the accuracy-oriented default.


## Validation

- 573 daemon tests, 43 app tests, and 200 Go tests pass. The 5 live Go tests now skip by default, independently of whether a user daemon exists.
- Xcode 27 test output uses new symbols. `make test-daemon` now relies on Swift's real exit status instead of matching old glyphs and potentially masking failures.
- A signed release TUI with its own daemon transcribed real Korean audio, displayed both MIC and SYS meters, produced no new summary/topic records, and shut down its daemon on quit.
- Converter regression tests cover 480-, 1,024-, and 4,096-frame chunks with complete output duration. The macOS 26 fallback was compiled, but this machine only exercised macOS 27 at runtime.
- Existing upstream Swift deprecation/test warnings remain; the release signature validates.
