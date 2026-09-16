package db

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

func TestTranscriptDeltaIncludesLateAudioAndAdvancesPastDuplicates(t *testing.T) {
	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	_, err = raw.Exec(`CREATE TABLE sessions(id TEXT PRIMARY KEY,status TEXT);
        CREATE TABLE segments(sessionId TEXT,sequenceNumber INTEGER,text TEXT,captured_at REAL,duplicate_of TEXT);
        INSERT INTO sessions VALUES ('s','active');
        INSERT INTO segments VALUES ('s',1,'old',100,NULL),('s',2,'older audio, finalized late',50,NULL),('s',3,'duplicate',110,'original');`)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(raw)
	got, err := store.ReadTranscriptDelta(context.Background(), "s", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "older audio, finalized late" || got.NextSequence != 3 || got.HasMore {
		t.Fatalf("lost late text or did not advance: %+v", got)
	}
	_, err = raw.Exec(`INSERT INTO segments VALUES ('s',4,'newly committed',40,NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	got, err = store.ReadTranscriptDelta(context.Background(), "s", got.NextSequence, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "newly committed" || got.NextSequence != 4 || got.HasMore {
		t.Fatalf("new append was missed: %+v", got)
	}
}

func TestTranscriptDeltaDuplicateOnlyTailIsEmpty(t *testing.T) {
	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	_, err = raw.Exec(`CREATE TABLE sessions(id TEXT PRIMARY KEY,status TEXT);
        CREATE TABLE segments(sessionId TEXT,sequenceNumber INTEGER,text TEXT,duplicate_of TEXT);
        INSERT INTO sessions VALUES ('s','active');
        INSERT INTO segments VALUES ('s',1,'duplicate','x');`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewStore(raw).ReadTranscriptDelta(context.Background(), "s", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "" || got.NextSequence != 1 || got.HasMore {
		t.Fatalf("wrong duplicate-only result: %+v", got)
	}
}
