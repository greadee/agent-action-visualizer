package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/security"
	"github.com/greadee/agent-action-visualizer/internal/session"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct{ db *sql.DB }

type SessionSummary struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	Status    string     `json:"status"`
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		var version int
		if _, err := fmt.Sscanf(entry.Name(), "%03d_", &version); err != nil {
			continue
		}
		var count int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=?", version).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		script, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(script)); err == nil {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)", version, time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version, err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveEvent(ctx context.Context, event protocol.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	event = security.SanitizeEvent(event)
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO events(event_id,session_id,timestamp,event_type,path,confidence,event_json) VALUES(?,?,?,?,?,?,?)`, event.EventID, event.SessionID, event.Timestamp.UTC().Format(time.RFC3339Nano), event.EventType, event.Path, event.SourceConfidence, string(payload))
	return err
}

// EnsureProject records the local project identity required by session rows.
func (s *Store) EnsureProject(ctx context.Context, id, root string) error {
	if id == "" || root == "" {
		return errors.New("project id and root are required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `INSERT INTO projects(id,root,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET root=excluded.root,updated_at=excluded.updated_at`, id, root, now, now)
	return err
}

func (s *Store) Events(ctx context.Context, sessionID string) ([]protocol.Event, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT event_json FROM events WHERE session_id=? ORDER BY sequence", sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []protocol.Event
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var event protocol.Event
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (s *Store) SaveSession(ctx context.Context, state session.State, projectID string) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	status := "active"
	var stopped interface{}
	if state.Paused {
		status = "paused"
	}
	if state.StoppedAt != nil {
		status = "stopped"
		stopped = state.StoppedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions(id,project_id,started_at,stopped_at,status,state_json) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET stopped_at=excluded.stopped_at,status=excluded.status,state_json=excluded.state_json`, state.SessionID, nullString(projectID), state.StartedAt.UTC().Format(time.RFC3339Nano), stopped, status, string(payload))
	return err
}

func (s *Store) Session(ctx context.Context, id string) (session.State, error) {
	var payload string
	if err := s.db.QueryRowContext(ctx, "SELECT state_json FROM sessions WHERE id=?", id).Scan(&payload); err != nil {
		return session.State{}, err
	}
	var state session.State
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return session.State{}, err
	}
	return state, nil
}

// Sessions returns persisted session metadata only, newest first.
func (s *Store) Sessions(ctx context.Context, projectID string) ([]SessionSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,COALESCE(project_id,''),started_at,stopped_at,status FROM sessions WHERE project_id=? ORDER BY started_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var item SessionSummary
		var started string
		var stopped sql.NullString
		if err := rows.Scan(&item.ID, &item.ProjectID, &started, &stopped, &item.Status); err != nil {
			return nil, err
		}
		var err error
		if item.StartedAt, err = time.Parse(time.RFC3339Nano, started); err != nil {
			return nil, err
		}
		if stopped.Valid {
			value, err := time.Parse(time.RFC3339Nano, stopped.String)
			if err != nil {
				return nil, err
			}
			item.StoppedAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecoverActiveSessions(ctx context.Context, at time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,state_json FROM sessions WHERE status IN ('active','paused')")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type item struct{ id, payload string }
	var items []item
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.id, &x.payload); err != nil {
			return 0, err
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, x := range items {
		var state session.State
		if err := json.Unmarshal([]byte(x.payload), &state); err != nil {
			return 0, err
		}
		state.StoppedAt = &at
		payload, _ := json.Marshal(state)
		if _, err := s.db.ExecContext(ctx, "UPDATE sessions SET stopped_at=?,status='recovered',state_json=? WHERE id=?", at.UTC().Format(time.RFC3339Nano), string(payload), x.id); err != nil {
			return 0, err
		}
	}
	return len(items), nil
}

func (s *Store) Tables(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
func nullString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
