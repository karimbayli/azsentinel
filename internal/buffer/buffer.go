package buffer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// Queue is a SQLite-backed local buffer for probe results.
// When the central server is unreachable, results are stored here
// and flushed in chronological order when connectivity resumes.
type Queue struct {
	db      *sql.DB
	maxSize int
	logger  *zap.Logger
	mu      sync.Mutex
}

// New creates a new disk-backed buffer queue.
func New(dbPath string, maxSize int, logger *zap.Logger) (*Queue, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create buffer dir %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}

	// Create buffer table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS buffer (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			payload    TEXT NOT NULL
		)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buffer table: %w", err)
	}

	return &Queue{
		db:      db,
		maxSize: maxSize,
		logger:  logger,
	}, nil
}

// Close closes the underlying SQLite database.
func (q *Queue) Close() error {
	return q.db.Close()
}

// Push adds a batch of probe results to the local buffer.
func (q *Queue) Push(batch models.ProbeBatch) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	payload, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	_, err = q.db.Exec(
		`INSERT INTO buffer (created_at, payload) VALUES (?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano),
		string(payload),
	)
	if err != nil {
		return fmt.Errorf("insert buffer: %w", err)
	}

	// Enforce max size — drop oldest entries
	if err := q.enforceMaxSize(); err != nil {
		q.logger.Error("failed to enforce buffer max size", zap.Error(err))
	}

	return nil
}

// Depth returns the number of buffered batches.
func (q *Queue) Depth() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	var count int
	err := q.db.QueryRow(`SELECT COUNT(*) FROM buffer`).Scan(&count)
	if err != nil {
		q.logger.Error("failed to query buffer depth", zap.Error(err))
		return 0
	}
	return count
}

// Peek returns the oldest N batches without removing them.
func (q *Queue) Peek(n int) ([]BufferedItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	rows, err := q.db.Query(
		`SELECT id, created_at, payload FROM buffer ORDER BY id ASC LIMIT ?`, n)
	if err != nil {
		return nil, fmt.Errorf("query buffer: %w", err)
	}
	defer rows.Close()

	var items []BufferedItem
	for rows.Next() {
		var item BufferedItem
		var payload string
		if err := rows.Scan(&item.ID, &item.CreatedAt, &payload); err != nil {
			return nil, fmt.Errorf("scan buffer row: %w", err)
		}
		if err := json.Unmarshal([]byte(payload), &item.Batch); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Remove deletes buffered items by their IDs after successful flush.
func (q *Queue) Remove(ids []int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.Prepare(`DELETE FROM buffer WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare delete: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			tx.Rollback()
			return fmt.Errorf("delete id %d: %w", id, err)
		}
	}

	return tx.Commit()
}

// enforceMaxSize drops the oldest entries if over capacity.
func (q *Queue) enforceMaxSize() error {
	var count int
	if err := q.db.QueryRow(`SELECT COUNT(*) FROM buffer`).Scan(&count); err != nil {
		return err
	}
	if count <= q.maxSize {
		return nil
	}

	excess := count - q.maxSize
	_, err := q.db.Exec(`DELETE FROM buffer WHERE id IN (SELECT id FROM buffer ORDER BY id ASC LIMIT ?)`, excess)
	if err != nil {
		return fmt.Errorf("trim buffer: %w", err)
	}

	q.logger.Warn("buffer overflow, dropped oldest entries",
		zap.Int("dropped", excess),
		zap.Int("remaining", q.maxSize))
	return nil
}

// BufferedItem is a single buffered probe batch with metadata.
type BufferedItem struct {
	ID        int64             `json:"id"`
	CreatedAt string            `json:"created_at"`
	Batch     models.ProbeBatch `json:"batch"`
}
