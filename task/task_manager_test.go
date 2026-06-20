package task

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupExpiredTask(t *testing.T) {
	tmpDir := t.TempDir()

	ttl := 2 * time.Second
	interval := 500 * time.Millisecond

	tm := NewManagerWithCleanup(ttl, interval)

	fpath := filepath.Join(tmpDir, "test_export.xlsx")
	if err := os.WriteFile(fpath, []byte("fake excel"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	task := tm.Create()
	tm.SetRunning(task.ID, 100)
	tm.SetCompleted(task.ID, "test_export.xlsx", fpath)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		t.Fatalf("file should exist before cleanup")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartCleanup(ctx)

	time.Sleep(4 * time.Second)

	if _, err := os.Stat(fpath); !os.IsNotExist(err) {
		t.Errorf("file should have been deleted after TTL expired")
	}

	_, ok := tm.Get(task.ID)
	if ok {
		t.Errorf("task should have been removed from manager after TTL expired")
	}
}

func TestCleanupKeepsActiveTask(t *testing.T) {
	tmpDir := t.TempDir()

	ttl := 5 * time.Second
	interval := 500 * time.Millisecond

	tm := NewManagerWithCleanup(ttl, interval)

	fpath := filepath.Join(tmpDir, "active_export.xlsx")
	if err := os.WriteFile(fpath, []byte("fake excel"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	task := tm.Create()
	tm.SetRunning(task.ID, 100)
	tm.SetCompleted(task.ID, "active_export.xlsx", fpath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartCleanup(ctx)

	time.Sleep(1 * time.Second)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		t.Errorf("file should still exist (within TTL)")
	}

	_, ok := tm.Get(task.ID)
	if !ok {
		t.Errorf("task should still be accessible (within TTL)")
	}
}

func TestCleanupPendingTaskNotRemoved(t *testing.T) {
	ttl := 1 * time.Second
	interval := 500 * time.Millisecond

	tm := NewManagerWithCleanup(ttl, interval)

	task := tm.Create()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartCleanup(ctx)

	time.Sleep(3 * time.Second)

	_, ok := tm.Get(task.ID)
	if !ok {
		t.Errorf("pending task should not be cleaned up")
	}
}
