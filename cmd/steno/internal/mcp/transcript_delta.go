package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jwulff/steno/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"math"
	"strings"
)

func registerTranscriptDelta(s *server.MCPServer, store *db.Store) {
	tool := mcp.NewTool("read_transcript",
		mcp.WithDescription("Preferred compact transcript feed after choosing a session. Returns only text, next_sequence and has_more. Pass next_sequence as after_sequence next time; after_sequence defaults to 0. Text is finalized, non-duplicate data in arrival/sequence order, so late-finalized audio is not skipped. No timestamps, IDs or sources per sentence. No shared consumption state: every caller keeps its own cursor. A session_status field is added only when the session is no longer active; use get_overview to find a new session then. Existing text can later be deduplicated; this feed does not send corrections."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID, discovered once via get_overview/list_sessions")),
		mcp.WithNumber("after_sequence", mcp.Min(0), mcp.Max(9007199254740991), mcp.Description("Last returned next_sequence for this session; integer >= 0, default 0")),
		mcp.WithNumber("limit", mcp.Min(1), mcp.Max(500), mcp.Description("Maximum new segments per response; integer 1..500, default 100")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		for key := range req.GetArguments() {
			if key != "session_id" && key != "after_sequence" && key != "limit" {
				return mcp.NewToolResultError("unknown argument " + key + "; use session_id, after_sequence and limit"), nil
			}
		}
		session, err := req.RequireString("session_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if strings.TrimSpace(session) == "" {
			return mcp.NewToolResultError("session_id must not be empty"), nil
		}
		after, err := deltaInt(req, "after_sequence", 0, 0, 9007199254740991)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit, err := deltaInt(req, "limit", 100, 1, 500)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := store.ReadTranscriptDelta(ctx, session, after, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}

func deltaInt(req mcp.CallToolRequest, name string, fallback, minValue, maxValue int) (int, error) {
	value, ok := req.GetArguments()[name]
	if !ok {
		return fallback, nil
	}
	var n float64
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	default:
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || n < float64(minValue) || n > float64(maxValue) {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minValue, maxValue)
	}
	return int(n), nil
}
