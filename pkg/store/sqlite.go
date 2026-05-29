package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS segments (
    id          INTEGER PRIMARY KEY,
    start_prime INTEGER NOT NULL,
    end_prime   INTEGER NOT NULL,
    count       INTEGER NOT NULL,
    algorithm   TEXT,
    elapsed_ms  INTEGER,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS benchmarks (
    id           INTEGER PRIMARY KEY,
    algorithm    TEXT NOT NULL,
    n_value      INTEGER,
    limit_value  INTEGER,
    elapsed_ms   INTEGER,
    primes_found INTEGER,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) RecordSegment(ctx context.Context, start, end, count uint64, algorithm string, elapsed time.Duration) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO segments (start_prime, end_prime, count, algorithm, elapsed_ms) VALUES (?, ?, ?, ?, ?)`,
		start, end, count, algorithm, elapsed.Milliseconds())
	return err
}

func (s *SQLiteStore) RecordBenchmark(ctx context.Context, algorithm string, n, limit, elapsedMs, primesFound uint64) error {
	var nVal, limitVal *uint64
	if n > 0 {
		nVal = &n
	}
	if limit > 0 {
		limitVal = &limit
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO benchmarks (algorithm, n_value, limit_value, elapsed_ms, primes_found) VALUES (?, ?, ?, ?, ?)`,
		algorithm, nVal, limitVal, elapsedMs, primesFound)
	return err
}

type SegmentInfo struct {
	ID        int64
	Start     uint64
	End       uint64
	Count     uint64
	Algorithm string
	ElapsedMs int64
	CreatedAt string
}

func (s *SQLiteStore) ListSegments(ctx context.Context) ([]SegmentInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, start_prime, end_prime, count, algorithm, elapsed_ms, created_at FROM segments ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segs []SegmentInfo
	for rows.Next() {
		var seg SegmentInfo
		if err := rows.Scan(&seg.ID, &seg.Start, &seg.End, &seg.Count, &seg.Algorithm, &seg.ElapsedMs, &seg.CreatedAt); err != nil {
			return nil, err
		}
		segs = append(segs, seg)
	}
	return segs, rows.Err()
}

type BenchmarkInfo struct {
	ID          int64
	Algorithm   string
	NValue      *uint64
	LimitValue  *uint64
	ElapsedMs   int64
	PrimesFound uint64
	CreatedAt   string
}

func (s *SQLiteStore) ListBenchmarks(ctx context.Context) ([]BenchmarkInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, algorithm, n_value, limit_value, elapsed_ms, primes_found, created_at FROM benchmarks ORDER BY id DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bencs []BenchmarkInfo
	for rows.Next() {
		var b BenchmarkInfo
		if err := rows.Scan(&b.ID, &b.Algorithm, &b.NValue, &b.LimitValue, &b.ElapsedMs, &b.PrimesFound, &b.CreatedAt); err != nil {
			return nil, err
		}
		bencs = append(bencs, b)
	}
	return bencs, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
