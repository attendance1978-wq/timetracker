package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Session struct {
	ID        string     `json:"id"`
	Task      string     `json:"task"`
	Project   string     `json:"project,omitempty"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

func (s *Session) Duration() time.Duration {
	if s.EndTime == nil {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}
func (s *Session) IsActive() bool { return s.EndTime == nil }

type dbData struct{ Sessions []Session `json:"sessions"` }

type Store struct {
	path string
	db   dbData
}

// New creates a Store using the default path ~/.timetracker/data.json
func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".timetracker")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return NewAt(filepath.Join(dir, "data.json"))
}

// NewAt creates a Store backed by a specific file path (useful for testing)
func NewAt(path string) (*Store, error) {
	s := &Store{path: path}
	return s, s.load()
}

// InjectSession inserts a completed session directly — used in tests
func (s *Store) InjectSession(task, project string, start, end time.Time) {
	s.db.Sessions = append(s.db.Sessions, Session{
		ID:        newID(),
		Task:      task,
		Project:   project,
		StartTime: start,
		EndTime:   &end,
	})
	s.save()
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &s.db)
}

func (s *Store) save() error {
	raw, _ := json.MarshalIndent(s.db, "", "  ")
	return os.WriteFile(s.path, raw, 0644)
}

func newID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) Start(task, project string) (*Session, error) {
	if a := s.Active(); a != nil {
		return nil, fmt.Errorf("session already running: %q — run 'track stop' first", a.Task)
	}
	sess := Session{ID: newID(), Task: task, Project: project, StartTime: time.Now()}
	s.db.Sessions = append(s.db.Sessions, sess)
	return &sess, s.save()
}

func (s *Store) Stop() (*Session, error) {
	for i := range s.db.Sessions {
		if s.db.Sessions[i].EndTime == nil {
			now := time.Now()
			s.db.Sessions[i].EndTime = &now
			if err := s.save(); err != nil {
				return nil, err
			}
			return &s.db.Sessions[i], nil
		}
	}
	return nil, fmt.Errorf("no active session — run 'track start <task>' to begin")
}

func (s *Store) Active() *Session {
	for i := range s.db.Sessions {
		if s.db.Sessions[i].EndTime == nil {
			return &s.db.Sessions[i]
		}
	}
	return nil
}

func (s *Store) ForDate(t time.Time) []Session {
	y, m, d := t.Date()
	var out []Session
	for _, sess := range s.db.Sessions {
		sy, sm, sd := sess.StartTime.Date()
		if sy == y && sm == m && sd == d {
			out = append(out, sess)
		}
	}
	return out
}

func (s *Store) ForWeek(t time.Time) []Session {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	start := t.AddDate(0, 0, -(wd-1))
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end := start.AddDate(0, 0, 7)
	var out []Session
	for _, sess := range s.db.Sessions {
		if !sess.StartTime.Before(start) && sess.StartTime.Before(end) {
			out = append(out, sess)
		}
	}
	return out
}

func (s *Store) ForMonth(t time.Time) []Session {
	y, m, _ := t.Date()
	var out []Session
	for _, sess := range s.db.Sessions {
		sy, sm, _ := sess.StartTime.Date()
		if sy == y && sm == m {
			out = append(out, sess)
		}
	}
	return out
}

func (s *Store) Delete(id string) error {
	for i, sess := range s.db.Sessions {
		if sess.ID == id {
			s.db.Sessions = append(s.db.Sessions[:i], s.db.Sessions[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("session %q not found", id)
}

func (s *Store) Edit(id, task, project string) (*Session, error) {
	for i := range s.db.Sessions {
		if s.db.Sessions[i].ID == id {
			if task != "" {
				s.db.Sessions[i].Task = task
			}
			if project != "" {
				s.db.Sessions[i].Project = project
			}
			return &s.db.Sessions[i], s.save()
		}
	}
	return nil, fmt.Errorf("session %q not found", id)
}
