package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tommitoan/personal-ops-hub/internal/core"
)

// Store persists cards, history, and incident transitions.
type Store struct{ Pool *pgxpool.Pool }

// Open connects to Postgres and applies the small idempotent schema.
func Open(ctx context.Context, dsn string) (*Store, error) {
	p, e := pgxpool.New(ctx, dsn)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	if _, e = p.Exec(ctx, schema); e != nil {
		p.Close()
		return nil, e
	}
	return &Store{p}, nil
}

const schema = `CREATE TABLE IF NOT EXISTS cards (key text PRIMARY KEY, body jsonb NOT NULL, updated_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS observations (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, key text NOT NULL, status text NOT NULL, summary text NOT NULL, observed_at timestamptz NOT NULL);
CREATE INDEX IF NOT EXISTS observations_key_time ON observations (key, observed_at DESC);
CREATE TABLE IF NOT EXISTS incidents (key text PRIMARY KEY, failures integer NOT NULL, open boolean NOT NULL, changed_at timestamptz NOT NULL);`

// SaveCard updates a snapshot and appends a bounded-history observation.
func (s *Store) SaveCard(ctx context.Context, c core.Card) error {
	b, e := json.Marshal(c)
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `INSERT INTO cards(key,body,updated_at) VALUES($1,$2,$3) ON CONFLICT(key) DO UPDATE SET body=$2,updated_at=$3`, c.Key, b, c.UpdatedAt); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO observations(key,status,summary,observed_at) VALUES($1,$2,$3,$4)`, c.Key, c.Status, c.Summary, c.UpdatedAt); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// Cards returns the most recent observation for each key.
func (s *Store) Cards(ctx context.Context) ([]core.Card, error) {
	rows, e := s.Pool.Query(ctx, `SELECT body FROM cards ORDER BY key`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []core.Card{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		var c core.Card
		if e = json.Unmarshal(b, &c); e != nil {
			return nil, e
		}
		if !c.ExpiresAt.IsZero() && time.Now().After(c.ExpiresAt) {
			c.Status = "stale"
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateIncident serializes a per-target transition and returns its event.
func (s *Store) UpdateIncident(ctx context.Context, key string, healthy bool, threshold int, now time.Time) (string, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `INSERT INTO incidents(key,failures,open,changed_at) VALUES($1,0,false,$2) ON CONFLICT DO NOTHING`, key, now); e != nil {
		return "", e
	}
	var st core.IncidentState
	st.Key = key
	if e = tx.QueryRow(ctx, `SELECT failures,open,changed_at FROM incidents WHERE key=$1 FOR UPDATE`, key).Scan(&st.ConsecutiveFailures, &st.Open, &st.ChangedAt); e != nil {
		return "", e
	}
	st, event := core.Transition(st, healthy, threshold, now)
	if _, e = tx.Exec(ctx, `UPDATE incidents SET failures=$2,open=$3,changed_at=$4 WHERE key=$1`, key, st.ConsecutiveFailures, st.Open, st.ChangedAt); e != nil {
		return "", e
	}
	return event, tx.Commit(ctx)
}

// History returns recent observations for one configured key.
func (s *Store) History(ctx context.Context, key string, limit int) ([]core.Card, error) {
	if limit < 1 || limit > 200 {
		return nil, errors.New("limit out of range")
	}
	rows, e := s.Pool.Query(ctx, `SELECT status,summary,observed_at FROM observations WHERE key=$1 ORDER BY observed_at DESC LIMIT $2`, key, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []core.Card{}
	for rows.Next() {
		var c core.Card
		c.Key = key
		if e = rows.Scan(&c.Status, &c.Summary, &c.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Prune deletes observations outside retention while preserving latest cards.
func (s *Store) Prune(ctx context.Context, days int) error {
	if days < 1 {
		return fmt.Errorf("invalid retention")
	}
	_, e := s.Pool.Exec(ctx, `DELETE FROM observations WHERE observed_at < now() - ($1 * interval '1 day')`, days)
	return e
}
