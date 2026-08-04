package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/session"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestMigrationsEventPersistenceAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aav.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tables, err := store.Tables(ctx)
	if err != nil || len(tables) < 10 {
		t.Fatalf("tables=%v err=%v", tables, err)
	}
	event := protocol.Event{SchemaVersion: protocol.SchemaVersion, EventID: "evt-1", SessionID: "s", EventType: protocol.EventFileRead, SourceType: protocol.SourceNativeHook, SourceConfidence: protocol.ConfidenceExact, Timestamp: time.Now().UTC(), Path: "a.go"}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	events, err := store.Events(ctx, "s")
	if err != nil || len(events) != 1 || events[0].Path != "a.go" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestSessionRecovery(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	started := time.Unix(1000, 0).UTC()
	state := session.State{SessionID: "s", StartedAt: started, ActivePath: "a.go"}
	if err := store.SaveSession(ctx, state, ""); err != nil {
		t.Fatal(err)
	}
	stop := started.Add(time.Minute)
	count, err := store.RecoverActiveSessions(ctx, stop)
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	recovered, err := store.Session(ctx, "s")
	if err != nil || recovered.StoppedAt == nil || !recovered.StoppedAt.Equal(stop) {
		t.Fatalf("state=%+v err=%v", recovered, err)
	}
}

func TestSessionsFiltersByProjectAndReturnsMetadata(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	started := time.Unix(1_000, 0).UTC()
	if err := store.EnsureProject(ctx, "project-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureProject(ctx, "project-b", "project-b"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSession(ctx, session.State{SessionID: "a", StartedAt: started}, "project-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSession(ctx, session.State{SessionID: "b", StartedAt: started.Add(time.Second)}, "project-b"); err != nil {
		t.Fatal(err)
	}
	sessions, err := store.Sessions(ctx, "project-a")
	if err != nil || len(sessions) != 1 || sessions[0].ID != "a" || !sessions[0].StartedAt.Equal(started) {
		t.Fatalf("sessions=%#v err=%v", sessions, err)
	}
}
