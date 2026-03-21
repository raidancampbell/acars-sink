package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/raidancampbell/acars-sink/internal/decoder"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store will encapsulate SQLite writes.
type Store struct {
	Path       string
	db         *sql.DB
	insertStmt *sql.Stmt
	insertParsedStmt *sql.Stmt
}

func ensureColumn(db *sql.DB, table string, column string, columnDef string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		cid       int
		name      string
		typeName  string
		notNull   int
		dfltValue sql.NullString
		pk        int
	)

	for rows.Next() {
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, columnDef))
	return err
}

func NewStore(path string) *Store {
	return &Store{Path: path}
}

// Init opens DB and prepares schema and insert statement.
func (s *Store) Init() error {
	if s.Path == "" {
		return errors.New("storage path is required")
	}

	db, err := sql.Open("sqlite", s.Path)
	if err != nil {
		return err
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return err
	}

	if err := ensureColumn(db, "messages", "source", "TEXT NOT NULL DEFAULT 'acars'"); err != nil {
		_ = db.Close()
		return err
	}
	if err := ensureColumn(db, "parsed_messages", "source", "TEXT NOT NULL DEFAULT 'acars'"); err != nil {
		_ = db.Close()
		return err
	}

	stmt, err := db.Prepare(`
INSERT INTO messages (received_at, source, raw_json, aircraft, flight, message_type, station)
VALUES (?, ?, ?, ?, ?, ?, ?);
`)
	if err != nil {
		_ = db.Close()
		return err
	}

	parsedStmt, err := db.Prepare(`
INSERT INTO parsed_messages (
  received_at,
  source,
  aircraft,
  flight,
  message_type,
  station,
  timestamp,
  label,
  message,
  text,
  channel,
  registration,
  icao
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`)
	if err != nil {
		_ = stmt.Close()
		_ = db.Close()
		return err
	}

	s.db = db
	s.insertStmt = stmt
	s.insertParsedStmt = parsedStmt
	return nil
}

func (s *Store) Insert(ctx context.Context, receivedAt time.Time, source string, rawJSON string, msg decoder.Message) error {
	if s.insertStmt == nil {
		return errors.New("storage not initialized")
	}

	_, err := s.insertStmt.ExecContext(
		ctx,
		receivedAt.UTC().Format(time.RFC3339Nano),
		source,
		rawJSON,
		msg.Aircraft,
		msg.Flight,
		msg.Type,
		msg.Station,
	)
	if err != nil {
		return err
	}

	if s.insertParsedStmt == nil {
		return errors.New("parsed storage not initialized")
	}

	_, err = s.insertParsedStmt.ExecContext(
		ctx,
		receivedAt.UTC().Format(time.RFC3339Nano),
		source,
		msg.Aircraft,
		msg.Flight,
		msg.Type,
		msg.Station,
		string(msg.Timestamp),
		msg.Label,
		msg.Message,
		msg.Text,
		string(msg.Channel),
		msg.Registration,
		msg.ICAO,
	)
	return err
}

func (s *Store) Close() error {
	if s.insertStmt != nil {
		_ = s.insertStmt.Close()
	}
	if s.insertParsedStmt != nil {
		_ = s.insertParsedStmt.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
