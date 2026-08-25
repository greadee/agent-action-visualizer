package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestSaveEventDoesNotRetainSourceContentOrSecrets(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	const secret = "p9s3-private-source-marker"
	event := protocol.Event{
		SchemaVersion: protocol.SchemaVersion, EventID: "privacy", SessionID: "session",
		EventType: protocol.EventCommandCompleted, SourceType: protocol.SourceWrapper,
		SourceConfidence: protocol.ConfidenceExact, Timestamp: time.Unix(1, 0).UTC(),
		Command: "runner --token " + secret, ActionLabel: "token=" + secret,
		Metadata: map[string]interface{}{
			"access_sequence": 3,
			"prompt":          "source " + secret,
			"tool_response":   secret,
			"environment":     map[string]interface{}{"API_KEY": secret},
			"reason":          "password=" + secret,
		},
	}
	if err := store.SaveEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	events, err := store.Events(context.Background(), "session")
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	payload, err := json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), secret) || events[0].Command != "" {
		t.Fatalf("sensitive payload was retained: %s", payload)
	}
	if _, found := events[0].Metadata["prompt"]; found {
		t.Fatalf("unsupported metadata was retained: %#v", events[0].Metadata)
	}
	if events[0].Metadata["access_sequence"] != float64(3) {
		t.Fatalf("required metadata was lost: %#v", events[0].Metadata)
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
	state := session.State{
		SessionID: "s", StartedAt: started, ActivePath: "a.go",
		Intervals: []session.AccessInterval{{Sequence: 1, Path: "a.go", StartedAt: started}},
	}
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
	if recovered.Intervals[0].EndedAt == nil || !recovered.Intervals[0].EndedAt.Equal(stop) || recovered.Intervals[0].Duration != time.Minute {
		t.Fatalf("stale interval was not closed: %+v", recovered.Intervals[0])
	}
}

func TestSessionRecoveryCapsStaleIntervalAtIdleTimeout(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	started := time.Unix(1_000, 0).UTC()
	state := session.State{
		SessionID: "stale", StartedAt: started, ActivePath: "a.go",
		Intervals: []session.AccessInterval{{Sequence: 1, Path: "a.go", StartedAt: started}},
	}
	if err := store.SaveSession(context.Background(), state, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecoverActiveSessions(context.Background(), started.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.Session(context.Background(), "stale")
	if err != nil {
		t.Fatal(err)
	}
	want := started.Add(session.DefaultIdleTimeout)
	if recovered.StoppedAt == nil || !recovered.StoppedAt.Equal(want) || recovered.Intervals[0].Duration != session.DefaultIdleTimeout {
		t.Fatalf("recovered state = %+v", recovered)
	}
}

func TestSessionRecoveryIsAtomicWhenOneStateIsCorrupt(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	started := time.Unix(1_000, 0).UTC()
	if err := store.SaveSession(context.Background(), session.State{SessionID: "good", StartedAt: started}, ""); err != nil {
		t.Fatal(err)
	}
	const privatePayload = "private-corrupt-session"
	_, err = store.db.Exec(`INSERT INTO sessions(id,started_at,status,state_json) VALUES(?,?,?,?)`, "bad", started.Format(time.RFC3339Nano), "active", privatePayload)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RecoverActiveSessions(context.Background(), started.Add(time.Minute))
	if !errors.Is(err, ErrCorruptRecord) || strings.Contains(err.Error(), privatePayload) {
		t.Fatalf("recovery error = %v", err)
	}
	good, err := store.Session(context.Background(), "good")
	if err != nil {
		t.Fatal(err)
	}
	if good.StoppedAt != nil {
		t.Fatalf("valid session was partially recovered: %+v", good)
	}
}

func TestInterruptedMigrationRollsBackAndRetries(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	err = store.applyMigration(ctx, 99, "CREATE TABLE interrupted(value TEXT); INSERT INTO missing_table(value) VALUES('x');", time.Unix(1, 0))
	if err == nil {
		t.Fatal("interrupted migration unexpectedly succeeded")
	}
	var count int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='interrupted'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial schema survived rollback: count=%d err=%v", count, err)
	}
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=99").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed migration was marked applied: count=%d err=%v", count, err)
	}
	if err := store.applyMigration(ctx, 99, "CREATE TABLE interrupted(value TEXT);", time.Unix(2, 0)); err != nil {
		t.Fatalf("migration retry failed: %v", err)
	}
}

func TestOpenRecoveringQuarantinesCorruptDatabaseWithoutDataLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aav.db")
	marker := []byte("not a sqlite database: private recovery fixture")
	if err := os.WriteFile(path, marker, 0o600); err != nil {
		t.Fatal(err)
	}
	store, recovery, err := OpenRecovering(path, time.Unix(1_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if recovery.Code != "database_quarantined" || recovery.QuarantinePath == "" {
		t.Fatalf("recovery = %+v", recovery)
	}
	preserved, err := os.ReadFile(recovery.QuarantinePath)
	if err != nil || string(preserved) != string(marker) {
		t.Fatalf("quarantined evidence was not preserved: %q err=%v", preserved, err)
	}
	if _, err := store.Tables(context.Background()); err != nil {
		t.Fatalf("replacement journal unavailable: %v", err)
	}
}

func TestEventsRejectCorruptRowsWithoutExposingPayload(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	const privatePayload = "private-source-content"
	_, err = store.db.Exec(`INSERT INTO events(event_id,session_id,timestamp,event_type,path,confidence,event_json) VALUES(?,?,?,?,?,?,?)`, "corrupt", "s", time.Unix(1, 0).UTC().Format(time.RFC3339Nano), protocol.EventFileRead, "a.go", protocol.ConfidenceExact, privatePayload)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Events(context.Background(), "s")
	if !errors.Is(err, ErrCorruptRecord) || strings.Contains(err.Error(), privatePayload) {
		t.Fatalf("unsafe corruption diagnostic: %v", err)
	}
}

func TestEventsUseCanonicalOrderAndIgnoreDuplicateDelivery(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aav.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	base := time.Unix(1_000, 0).UTC()
	events := []protocol.Event{
		{SchemaVersion: protocol.SchemaVersion, EventID: "stop", SessionID: "s", EventType: protocol.EventSessionStopped, SourceType: protocol.SourceSynthetic, SourceConfidence: protocol.ConfidenceExact, Timestamp: base.Add(3 * time.Second)},
		{SchemaVersion: protocol.SchemaVersion, EventID: "later", SessionID: "s", EventType: protocol.EventFileRead, SourceType: protocol.SourceSynthetic, SourceConfidence: protocol.ConfidenceExact, Timestamp: base.Add(2 * time.Second), Path: "b.go"},
		{SchemaVersion: protocol.SchemaVersion, EventID: "start", SessionID: "s", EventType: protocol.EventSessionStarted, SourceType: protocol.SourceSynthetic, SourceConfidence: protocol.ConfidenceExact, Timestamp: base},
		{SchemaVersion: protocol.SchemaVersion, EventID: "earlier", SessionID: "s", EventType: protocol.EventFileRead, SourceType: protocol.SourceSynthetic, SourceConfidence: protocol.ConfidenceExact, Timestamp: base.Add(time.Second), Path: "a.go"},
	}
	for _, event := range append(events, events[1]) {
		if err := store.SaveEvent(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	ordered, err := store.Events(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 4 || ordered[0].EventID != "start" || ordered[1].EventID != "earlier" || ordered[2].EventID != "later" || ordered[3].EventID != "stop" {
		t.Fatalf("canonical events = %#v", ordered)
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
