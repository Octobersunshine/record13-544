package task

import (
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
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
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*Task),
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
