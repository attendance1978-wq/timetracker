package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"timetracker/store"
)

// newTestStore creates a Store backed by a temp dir so tests don't touch ~/.timetracker
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.NewAt(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("NewAt: %v", err)
	}
	return s
}

// ─── Start ────────────────────────────────────────────────────────────────────

func TestStart_CreatesSession(t *testing.T) {
	s := newTestStore(t)
	sess, err := s.Start("Write tests", "backend")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if sess.Task != "Write tests" {
		t.Errorf("Task = %q, want %q", sess.Task, "Write tests")
	}
	if sess.Project != "backend" {
		t.Errorf("Project = %q, want %q", sess.Project, "backend")
	}
	if sess.ID == "" {
		t.Error("ID should not be empty")
	}
	if sess.EndTime != nil {
		t.Error("EndTime should be nil for active session")
	}
}

func TestStart_NoProjectAllowed(t *testing.T) {
	s := newTestStore(t)
	sess, err := s.Start("Quick note", "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if sess.Project != "" {
		t.Errorf("Project = %q, want empty", sess.Project)
	}
}

func TestStart_RejectsDoubleStart(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Start("First task", ""); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	_, err := s.Start("Second task", "")
	if err == nil {
		t.Fatal("expected error when starting a second session, got nil")
	}
}

// ─── Stop ─────────────────────────────────────────────────────────────────────

func TestStop_EndsActiveSession(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Start("Do work", ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	sess, err := s.Stop()
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if sess.EndTime == nil {
		t.Error("EndTime should be set after Stop")
	}
	if sess.EndTime.Before(sess.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

func TestStop_ErrorWhenNoActive(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Stop()
	if err == nil {
		t.Fatal("expected error stopping with no active session")
	}
}

func TestStop_ClearsActiveSession(t *testing.T) {
	s := newTestStore(t)
	s.Start("Task", "")
	s.Stop()
	if s.Active() != nil {
		t.Error("Active() should return nil after Stop")
	}
}

// ─── Active ───────────────────────────────────────────────────────────────────

func TestActive_NilWhenEmpty(t *testing.T) {
	s := newTestStore(t)
	if s.Active() != nil {
		t.Error("Active() should be nil on empty store")
	}
}

func TestActive_ReturnsRunningSession(t *testing.T) {
	s := newTestStore(t)
	s.Start("Running task", "proj")
	a := s.Active()
	if a == nil {
		t.Fatal("Active() should not be nil")
	}
	if a.Task != "Running task" {
		t.Errorf("Active task = %q, want %q", a.Task, "Running task")
	}
}

// ─── ForDate ──────────────────────────────────────────────────────────────────

func TestForDate_ReturnsMatchingDay(t *testing.T) {
	s := newTestStore(t)
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	s.Start("Today task", "")
	s.Stop()
	// Manually inject a yesterday session
	s.InjectSession("Yesterday task", "", yesterday, yesterday.Add(30*time.Minute))

	todaySessions := s.ForDate(today)
	if len(todaySessions) != 1 {
		t.Errorf("ForDate(today) = %d sessions, want 1", len(todaySessions))
	}
	if todaySessions[0].Task != "Today task" {
		t.Errorf("got task %q, want %q", todaySessions[0].Task, "Today task")
	}

	yestSessions := s.ForDate(yesterday)
	if len(yestSessions) != 1 {
		t.Errorf("ForDate(yesterday) = %d sessions, want 1", len(yestSessions))
	}
}

func TestForDate_EmptyOnNoSessions(t *testing.T) {
	s := newTestStore(t)
	if got := s.ForDate(time.Now()); len(got) != 0 {
		t.Errorf("ForDate on empty store = %d sessions, want 0", len(got))
	}
}

// ─── ForWeek ──────────────────────────────────────────────────────────────────

func TestForWeek_IncludesThisWeek(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	// sessions within the week
	for i := 0; i < 3; i++ {
		day := now.AddDate(0, 0, -i)
		// only go back within the same ISO week
		if int(day.Weekday()) == 0 && i > 0 {
			break
		}
		s.InjectSession("Week task", "", day, day.Add(time.Hour))
	}
	// one session 10 days ago (outside week)
	old := now.AddDate(0, 0, -10)
	s.InjectSession("Old task", "", old, old.Add(time.Hour))

	sessions := s.ForWeek(now)
	for _, sess := range sessions {
		if sess.Task == "Old task" {
			t.Error("ForWeek returned a session from 10 days ago")
		}
	}
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestDelete_RemovesSession(t *testing.T) {
	s := newTestStore(t)
	sess, _ := s.Start("To delete", "")
	s.Stop()
	id := sess.ID

	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(s.ForDate(time.Now())) != 0 {
		t.Error("session should be gone after Delete")
	}
}

func TestDelete_ErrorOnBadID(t *testing.T) {
	s := newTestStore(t)
	if err := s.Delete("doesnotexist"); err == nil {
		t.Error("expected error deleting non-existent ID")
	}
}

// ─── Edit ─────────────────────────────────────────────────────────────────────

func TestEdit_UpdatesTask(t *testing.T) {
	s := newTestStore(t)
	sess, _ := s.Start("Old task", "proj")
	s.Stop()

	updated, err := s.Edit(sess.ID, "New task", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if updated.Task != "New task" {
		t.Errorf("Task = %q, want %q", updated.Task, "New task")
	}
	// project unchanged
	if updated.Project != "proj" {
		t.Errorf("Project = %q, want unchanged %q", updated.Project, "proj")
	}
}

func TestEdit_UpdatesProject(t *testing.T) {
	s := newTestStore(t)
	sess, _ := s.Start("Task", "old-proj")
	s.Stop()

	updated, err := s.Edit(sess.ID, "", "new-proj")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if updated.Project != "new-proj" {
		t.Errorf("Project = %q, want %q", updated.Project, "new-proj")
	}
	if updated.Task != "Task" {
		t.Errorf("Task should be unchanged, got %q", updated.Task)
	}
}

func TestEdit_ErrorOnBadID(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Edit("badid", "task", "")
	if err == nil {
		t.Error("expected error editing non-existent ID")
	}
}

// ─── Persistence ─────────────────────────────────────────────────────────────

func TestPersistence_ReloadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	s1, _ := store.NewAt(path)
	s1.Start("Persist me", "proj")
	s1.Stop()

	s2, err := store.NewAt(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	sessions := s2.ForDate(time.Now())
	if len(sessions) != 1 {
		t.Fatalf("after reload got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Task != "Persist me" {
		t.Errorf("Task = %q after reload, want %q", sessions[0].Task, "Persist me")
	}
}

// ─── Duration ─────────────────────────────────────────────────────────────────

func TestSession_Duration_Active(t *testing.T) {
	s := newTestStore(t)
	s.Start("Active task", "")
	a := s.Active()
	dur := a.Duration()
	if dur < 0 {
		t.Errorf("active session duration should be >= 0, got %v", dur)
	}
}

func TestSession_Duration_Completed(t *testing.T) {
	s := newTestStore(t)
	sess, _ := s.Start("Task", "")
	time.Sleep(10 * time.Millisecond)
	stopped, _ := s.Stop()

	if stopped.Duration() <= 0 {
		t.Errorf("completed session duration should be > 0, got %v", stopped.Duration())
	}
	_ = sess
}

// ─── IsActive ─────────────────────────────────────────────────────────────────

func TestSession_IsActive(t *testing.T) {
	s := newTestStore(t)
	s.Start("Task", "")
	a := s.Active()
	if !a.IsActive() {
		t.Error("IsActive() should be true for running session")
	}
	stopped, _ := s.Stop()
	if stopped.IsActive() {
		t.Error("IsActive() should be false after stop")
	}
}

// ─── IDs are unique ───────────────────────────────────────────────────────────

func TestStart_UniqueIDs(t *testing.T) {
	s := newTestStore(t)
	ids := make(map[string]bool)
	for i := 0; i < 20; i++ {
		sess, _ := s.Start("Task", "")
		if ids[sess.ID] {
			t.Errorf("duplicate ID generated: %s", sess.ID)
		}
		ids[sess.ID] = true
		s.Stop()
	}
}

// ─── File not found is not an error ──────────────────────────────────────────

func TestNewAt_MissingFileIsOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	s, err := store.NewAt(path)
	if err != nil {
		t.Fatalf("NewAt with missing file: %v", err)
	}
	if s.Active() != nil {
		t.Error("fresh store should have no active session")
	}
	_ = os.Remove(path) // clean up just in case
}
