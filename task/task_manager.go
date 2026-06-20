package task

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusExpired   Status = "expired"
)

type Task struct {
	ID          string    `json:"id"`
	Status      Status    `json:"status"`
	Progress    int       `json:"progress"`
	Total       int       `json:"total"`
	FileName    string    `json:"file_name,omitempty"`
	FilePath    string    `json:"-"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type Manager struct {
	mu             sync.RWMutex
	tasks          map[string]*Task
	ttl            time.Duration
	cleanupInterval time.Duration
}

func NewManager() *Manager {
	return &Manager{
		tasks:           make(map[string]*Task),
		ttl:            24 * time.Hour,
		cleanupInterval: 1 * time.Hour,
	}
}

func NewManagerWithCleanup(ttl, cleanupInterval time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Hour
	}
	return &Manager{
		tasks:           make(map[string]*Task),
		ttl:            ttl,
		cleanupInterval: cleanupInterval,
	}
}

func (m *Manager) Create() *Task {
	id := fmt.Sprintf("task_%d", time.Now().UnixNano())
	t := &Task{
		ID:        id,
		Status:    StatusPending,
		Progress:  0,
		Total:     0,
		CreatedAt: time.Now(),
	}
	m.mu.Lock()
	m.tasks[id] = t
	m.mu.Unlock()
	return t
}

func (m *Manager) Get(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

func (m *Manager) SetRunning(id string, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusRunning
		t.Total = total
	}
}

func (m *Manager) UpdateProgress(id string, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Progress = progress
	}
}

func (m *Manager) SetCompleted(id string, fileName, filePath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusCompleted
		t.FileName = fileName
		t.FilePath = filePath
		t.CompletedAt = time.Now()
		if t.Total > 0 {
			t.Progress = t.Total
		}
	}
}

func (m *Manager) SetFailed(id string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusFailed
		t.Error = errMsg
		t.CompletedAt = time.Now()
	}
}

func (m *Manager) Remove(id string) *Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return nil
	}
	delete(m.tasks, id)
	return t
}

func (m *Manager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()

		log.Printf("[cleanup] started: ttl=%s, interval=%s", m.ttl, m.cleanupInterval)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[cleanup] stopped")
				return
			case <-ticker.C:
				m.sweep()
			}
		}
	}()
}

func (m *Manager) sweep() {
	now := time.Now()
	var expired []*Task

	m.mu.RLock()
	for _, t := range m.tasks {
		if t.Status == StatusCompleted || t.Status == StatusFailed {
			if !t.CompletedAt.IsZero() && now.Sub(t.CompletedAt) > m.ttl {
				expired = append(expired, t)
			}
		}
	}
	m.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	for _, t := range expired {
		if t.FilePath != "" {
			if err := os.Remove(t.FilePath); err != nil && !os.IsNotExist(err) {
				log.Printf("[cleanup] failed to delete file %s: %v", t.FilePath, err)
				continue
			}
			log.Printf("[cleanup] deleted file: %s", t.FilePath)
		}

		m.mu.Lock()
		delete(m.tasks, t.ID)
		m.mu.Unlock()

		log.Printf("[cleanup] removed expired task: %s (status=%s, age=%s)",
			t.ID, t.Status, now.Sub(t.CompletedAt).Round(time.Second))
	}

	log.Printf("[cleanup] swept %d expired task(s)", len(expired))
}
