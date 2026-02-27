package buffer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"go.uber.org/zap"
)

func TestBufferQueue(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "sentinel-buffer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	logger := zap.NewNop()

	q, err := New(dbPath, 100, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Test initial depth
	if depth := q.Depth(); depth != 0 {
		t.Errorf("expected depth 0, got %d", depth)
	}

	// Test push
	batch := models.ProbeBatch{
		NodeID:    "node-test",
		Region:    "test",
		Timestamp: time.Now().UTC(),
		Results: []models.ProbeResult{
			{
				Time:       time.Now().UTC(),
				NodeID:     "node-test",
				TargetURL:  "https://example.com",
				TCPSuccess: true,
				HTTPStatus: 200,
			},
		},
	}

	if err := q.Push(batch); err != nil {
		t.Fatal(err)
	}

	// Test depth after push
	if depth := q.Depth(); depth != 1 {
		t.Errorf("expected depth 1, got %d", depth)
	}

	// Test peek
	items, err := q.Peek(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Batch.NodeID != "node-test" {
		t.Errorf("expected node_id 'node-test', got '%s'", items[0].Batch.NodeID)
	}
	if len(items[0].Batch.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(items[0].Batch.Results))
	}

	// Test remove
	if err := q.Remove([]int64{items[0].ID}); err != nil {
		t.Fatal(err)
	}
	if depth := q.Depth(); depth != 0 {
		t.Errorf("expected depth 0 after remove, got %d", depth)
	}
}

func TestBufferMaxSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sentinel-buffer-max-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	logger := zap.NewNop()

	maxSize := 5
	q, err := New(dbPath, maxSize, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	// Push more than max
	for i := 0; i < maxSize+3; i++ {
		batch := models.ProbeBatch{
			NodeID:    "node-test",
			Timestamp: time.Now().UTC(),
			Results:   []models.ProbeResult{{Time: time.Now().UTC()}},
		}
		if err := q.Push(batch); err != nil {
			t.Fatal(err)
		}
	}

	depth := q.Depth()
	if depth > maxSize {
		t.Errorf("expected depth <= %d, got %d", maxSize, depth)
	}
}
