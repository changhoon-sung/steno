package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadTranscriptReturnsOnlyNewTextAndCursor(t *testing.T) {
	s := testServer(t)
	text := callTool(t, s, "read_transcript", map[string]any{"session_id": "sess-2", "after_sequence": 1})
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["text"] != "Active session segment 2.\nActive session segment 3." || got["next_sequence"] != float64(3) || got["has_more"] != false {
		t.Fatalf("unexpected delta: %s", text)
	}
	if strings.Contains(text, "started_at") || strings.Contains(text, "session_id") {
		t.Fatal("repeated metadata leaked into compact feed")
	}
}

func TestReadTranscriptPagesAndReturnsEmptyWhenCaughtUp(t *testing.T) {
	s := testServer(t)
	var first, second, empty map[string]any
	json.Unmarshal([]byte(callTool(t, s, "read_transcript", map[string]any{"session_id": "sess-2", "limit": 2})), &first)
	if first["next_sequence"] != float64(2) || first["has_more"] != true {
		t.Fatalf("bad first page: %v", first)
	}
	json.Unmarshal([]byte(callTool(t, s, "read_transcript", map[string]any{"session_id": "sess-2", "after_sequence": 2, "limit": 2})), &second)
	if second["text"] != "Active session segment 3." || second["next_sequence"] != float64(3) || second["has_more"] != false {
		t.Fatalf("bad second page: %v", second)
	}
	json.Unmarshal([]byte(callTool(t, s, "read_transcript", map[string]any{"session_id": "sess-2", "after_sequence": 3})), &empty)
	if empty["text"] != "" || empty["next_sequence"] != float64(3) || empty["has_more"] != false {
		t.Fatalf("bad empty response: %v", empty)
	}
}

func TestReadTranscriptCursorsAreCallerOwned(t *testing.T) {
	s := testServer(t)
	args := map[string]any{"session_id": "sess-2", "after_sequence": 1}
	a := callTool(t, s, "read_transcript", args)
	b := callTool(t, s, "read_transcript", args)
	if a != b {
		t.Fatal("reading consumed another caller's transcript")
	}
}

func TestReadTranscriptRejectsInvalidAndUnknownArguments(t *testing.T) {
	s := testServer(t)
	for _, args := range []map[string]any{
		{"session_id": "sess-2", "since": "2026-01-01"},
		{"session_id": "sess-2", "after_sequence": -1},
		{"session_id": "sess-2", "after_sequence": 1.5},
		{"session_id": "sess-2", "after_sequence": "1"},
		{"session_id": "sess-2", "after_sequence": 99},
		{"session_id": "sess-2", "limit": 0},
		{"session_id": "sess-2", "limit": 501},
		{"session_id": "missing"},
		{"session_id": ""},
	} {
		text := callTool(t, s, "read_transcript", args)
		var got map[string]any
		if json.Unmarshal([]byte(text), &got) == nil && got["text"] != nil {
			t.Fatalf("accepted bad arguments: %v", args)
		}
	}
}

func TestReadTranscriptSignalsEndedSession(t *testing.T) {
	s := testServer(t)
	text := callTool(t, s, "read_transcript", map[string]any{"session_id": "sess-1", "after_sequence": 10})
	var got map[string]any
	json.Unmarshal([]byte(text), &got)
	if got["session_status"] != "completed" || got["text"] != "" {
		t.Fatalf("missing session boundary: %s", text)
	}
}
