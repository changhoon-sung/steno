# Compact incremental transcript feed

Add `read_transcript(session_id, after_sequence=0, limit=100)` for repeated context updates after choosing a session. Return a single text string plus `next_sequence` and `has_more`, without per-segment IDs, timestamps, confidence, or repeated session/source fields. Only inactive sessions add `session_status` to signal a boundary.

A read-only SQLite transaction pins the session status, sequence high-water mark, and returned rows to one snapshot. Pagination uses commit/sequence order, not audio timestamps or mutable offsets, so late-finalized audio is not missed. Known duplicates are filtered and their trailing sequence numbers can still advance the cursor. Cursors belong to callers and sessions; reads do not consume shared state. Reject invalid/future cursors, unknown arguments, and invalid page sizes explicitly.

This is an append feed, not a revision protocol: later deduplication or edits do not retract previously delivered text. The original metadata-rich `get_transcript` remains available. MCP instructions now prefer the compact feed after a one-time session lookup.

Version: 0.5.2-tui.5. Reconnect existing MCP server processes to load the new tool.


Validation: 837 local tests passed (5 live-daemon tests disabled). An actual stdio MCP client discovered the sixth read-only tool and read the production database without starting audio capture. A 20-segment comparison reduced response text from 7,260 to 1,731 UTF-8 bytes (76.2%); cursor continuation passed. Tests cover pagination, empty reads, independent callers, unknown/invalid arguments, ended sessions, late-finalized audio, and duplicate-only tails.
