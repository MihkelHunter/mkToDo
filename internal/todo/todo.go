// Package todo defines the core domain model and service interface.
// The Repository interface allows swapping storage backends (SQLite, Postgres, in-memory, etc.)
// without changing any other layer — important for adding a web UI later.
package todo

import "time"

// Priority levels for a task.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	default:
		return "Low"
	}
}

// Task is the central domain object.
type Task struct {
	ID          int64
	Title       string
	Description string
	Done        bool
	Priority    Priority
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository is the storage contract. Any backend (SQLite, Postgres, memory)
// must satisfy this interface. The desktop app and future web server both use
// this interface, never a concrete type.
type Repository interface {
	Create(t *Task) error
	List() ([]*Task, error)
	Update(t *Task) error
	Delete(id int64) error
	Close() error
}

// Service wraps the repository and holds business logic.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Add(title, description string, priority Priority) (*Task, error) {
	now := time.Now()
	t := &Task{
		Title:       title,
		Description: description,
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) All() ([]*Task, error) {
	return s.repo.List()
}

func (s *Service) Toggle(t *Task) error {
	t.Done = !t.Done
	t.UpdatedAt = time.Now()
	return s.repo.Update(t)
}

func (s *Service) Edit(t *Task, title, description string, priority Priority) error {
	t.Title = title
	t.Description = description
	t.Priority = priority
	t.UpdatedAt = time.Now()
	return s.repo.Update(t)
}

func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *Service) Close() error {
	return s.repo.Close()
}
