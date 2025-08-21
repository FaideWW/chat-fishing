package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/faideww/chat-fishing/internal/fish"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db             *sql.DB
	insertStmt     *sql.Stmt
	topStmt        *sql.Stmt
	topSpeciesStmt *sql.Stmt
}

func OpenSQLite(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create db path: %w", err)
	}

	// DSN notes:
	// - _pragma=busy_timeout sets a lock wait
	// - _pragma=journal_mode(WAL) enables the write-ahead log
	// - _pragma=synchronous(NORMAL) sets the disk synchronizing
	//	 mode to NORMAL (recommended with WAL enabled)
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", filepath.Clean(dbPath))

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	ins, err := db.Prepare(`
		INSERT INTO catches (guild_id, user_id, species_id, size_tenths, caught_at)
		VALUES (?,?,?,?,?)
	`)

	if err != nil {
		_ = db.Close()
		return nil, err
	}

	top, err := db.Prepare(`
		SELECT id, guild_id, user_id, species_id, size_tenths, caught_at
		FROM catches
		WHERE guild_id = ?
		ORDER BY size_tenths DESC, id DESC 
		LIMIT ?
	`)

	if err != nil {
		_ = ins.Close()
		_ = db.Close()
		return nil, err
	}

	topSpecies, err := db.Prepare(`
		SELECT id, guild_id, user_id, species_id, size_tenths, caught_at
		FROM catches
		WHERE guild_id = ? AND species_id = ?
		ORDER BY size_tenths DESC, id DESC 
		LIMIT ?
	`)

	if err != nil {
		_ = ins.Close()
		_ = top.Close()
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db, insertStmt: ins, topStmt: top, topSpeciesStmt: topSpecies}, nil

}

func (s *SQLiteStore) Close() error {
	if s.insertStmt != nil {
		_ = s.insertStmt.Close()
	}
	if s.topStmt != nil {
		_ = s.topStmt.Close()
	}
	if s.topSpeciesStmt != nil {
		_ = s.topSpeciesStmt.Close()
	}

	return s.db.Close()
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS catches (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id     BIGINT  NOT NULL,
			user_id      BIGINT  NOT NULL,
			species_id   INTEGER NOT NULL,
			size_tenths  INTEGER NOT NULL,
			caught_at    INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_leader_all
			ON catches (guild_id, size_tenths DESC, id DESC);

		CREATE INDEX IF NOT EXISTS idx_leader_species
			ON catches (guild_id, species_id, size_tenths DESC, id DESC);
	`)
	return err
}

func (s *SQLiteStore) Add(ctx context.Context, c fish.Catch) error {
	if s == nil || s.db == nil {
		return errors.New("store not initialized")
	}

	if c.CaughtAt.IsZero() {
		c.CaughtAt = time.Now()
	}

	sizeTenths := int64(math.Round(c.Size * 10.0))
	_, err := s.insertStmt.Exec(
		c.GuildId,
		c.UserId,
		c.SpeciesId,
		sizeTenths,
		c.CaughtAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) TopBySize(ctx context.Context, guildId int64, limit int) ([]fish.Catch, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store not initialized")
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.topStmt.Query(guildId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]fish.Catch, 0, limit)
	for rows.Next() {
		var (
			id, gid, uid int64
			spid         int
			sizeTenths   int64
			caughtUnix   int64
		)
		if err := rows.Scan(&id, &gid, &uid, &spid, &sizeTenths, &caughtUnix); err != nil {
			return nil, err
		}

		out = append(out, fish.Catch{
			Id:        id,
			GuildId:   gid,
			UserId:    uid,
			SpeciesId: fish.SpeciesId(spid),
			Size:      float64(sizeTenths) / 10.0,
			CaughtAt:  time.Unix(caughtUnix, 0).UTC(),
		})
	}

	return out, rows.Err()
}

func (s *SQLiteStore) TopBySizeGuildSpecies(ctx context.Context, guildId int64, speciesId fish.SpeciesId, limit int) ([]fish.Catch, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store not initialized")
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.topSpeciesStmt.Query(guildId, speciesId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]fish.Catch, 0, limit)
	for rows.Next() {
		var (
			id, gid, uid int64
			spid         int
			sizeTenths   int64
			caughtUnix   int64
		)
		if err := rows.Scan(&id, &gid, &uid, &spid, &sizeTenths, &caughtUnix); err != nil {
			return nil, err
		}

		out = append(out, fish.Catch{
			Id:        id,
			GuildId:   gid,
			UserId:    uid,
			SpeciesId: fish.SpeciesId(spid),
			Size:      float64(sizeTenths) / 10.0,
			CaughtAt:  time.Unix(caughtUnix, 0).UTC(),
		})
	}

	return out, rows.Err()
}
