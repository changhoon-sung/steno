package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TranscriptDelta is an append-oriented, caller-cursored view of finalized
// text. A read transaction pins the high-water mark and rows to one snapshot.
type TranscriptDelta struct {
	Text         string `json:"text"`
	NextSequence int    `json:"next_sequence"`
	HasMore      bool   `json:"has_more"`
	// Only emitted on a session boundary; active reads remain three fields.
	SessionStatus string `json:"session_status,omitempty"`
}

func (s *Store) ReadTranscriptDelta(ctx context.Context, sessionID string, after, limit int) (TranscriptDelta, error) {
	result := TranscriptDelta{NextSequence: after}
	if after < 0 || limit < 1 || limit > 500 {
		return result, fmt.Errorf("invalid transcript cursor or limit")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var status string
	var highWater int
	err = tx.QueryRowContext(ctx, `SELECT status, COALESCE((SELECT MAX(sequenceNumber) FROM segments WHERE sessionId=?),0) FROM sessions WHERE id=?`, sessionID, sessionID).Scan(&status, &highWater)
	if err == sql.ErrNoRows {
		return result, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return result, err
	}
	if after > highWater {
		return result, fmt.Errorf("after_sequence exceeds this session's latest sequence; use a separate cursor for each session")
	}
	if status != "active" {
		result.SessionStatus = status
	}
	rows, err := tx.QueryContext(ctx, `SELECT sequenceNumber,text FROM segments
        WHERE sessionId=? AND sequenceNumber>? AND sequenceNumber<=? AND duplicate_of IS NULL
        ORDER BY sequenceNumber ASC LIMIT ?`, sessionID, after, highWater, limit+1)
	if err != nil {
		return result, err
	}
	var lines []string
	count := 0
	for rows.Next() {
		var sequence int
		var text string
		if err := rows.Scan(&sequence, &text); err != nil {
			rows.Close()
			return result, err
		}
		if count == limit {
			result.HasMore = true
			break
		}
		count++
		result.NextSequence = sequence
		if text = strings.TrimSpace(text); text != "" {
			lines = append(lines, text)
		}
	}
	scanErr := rows.Err()
	rows.Close()
	if scanErr != nil {
		return result, scanErr
	}
	if !result.HasMore {
		result.NextSequence = highWater
	} // Advance past filtered duplicates too.
	result.Text = strings.Join(lines, "\n")
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
