package daemon

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestDaemonCrashDuringRecording tests if the daemon crashes during recording
// even without an event subscriber. This isolates whether the crash is related
// to event broadcasting or to SpeechAnalyzer itself.
func TestDaemonCrashDuringRecording(t *testing.T) {
	// Explicit opt-in: these tests can stop/restart the running daemon.
	if os.Getenv("STENO_LIVE_TESTS") != "1" {
		t.Skip("set STENO_LIVE_TESTS=1 only with a disposable daemon")
	}
	sockPath := SocketPath()
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Skip("daemon not running")
	}

	client, err := Connect(sockPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	// Start recording
	ensureIdle(t, client)

	resp, err := client.SendCommand(Command{Cmd: "start"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !resp.OK {
		t.Fatalf("start failed: %s", resp.Error)
	}
	fmt.Printf("Started recording: sessionId=%s\n", resp.SessionID)

	// Wait 5 seconds to let SpeechAnalyzer actually process audio
	fmt.Println("Waiting 5 seconds to let speech recognizer run...")
	time.Sleep(5 * time.Second)

	// Check if daemon is still alive by sending status
	resp, err = client.SendCommand(Command{Cmd: "status"})
	if err != nil {
		t.Fatalf("daemon crashed during recording (status failed): %v", err)
	}
	fmt.Printf("Still alive after 5s: recording=%v segments=%v\n", derefBool(resp.Recording), derefInt(resp.Segments))

	// Stop
	resp, err = client.SendCommand(Command{Cmd: "stop"})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	fmt.Printf("Stopped: recording=%v\n", derefBool(resp.Recording))
}
