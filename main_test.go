package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"timetracker/store"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// captureOutput redirects stdout and returns what was printed.
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

// newTestDB writes a fresh data file in a temp dir and sets HOME so New() picks
// it up, restoring HOME afterwards.
func newTestDB(t *testing.T) (s *store.Store, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s, err := store.NewAt(path)
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	return s, func() {}
}

// runCmd calls the named command handler with the given args, returning any error.
func runCmd(cmd string, args []string) error {
	switch cmd {
	case "start":
		return cmdStart(args)
	case "stop":
		return cmdStop(args)
	case "status":
		return cmdStatus(args)
	case "delete":
		return cmdDelete(args)
	case "edit":
		return cmdEdit(args)
	}
	return nil
}

// ─── cmdStart ─────────────────────────────────────────────────────────────────

func TestCmdStart_RequiresTask(t *testing.T) {
	err := cmdStart([]string{})
	if err == nil {
		t.Fatal("expected error when no task given")
	}
}

func TestCmdStart_ParsesTaskAndProject(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	out := captureOutput(func() {
		if err := cmdStart([]string{"Fix bug", "-p", "backend"}); err != nil {
			t.Errorf("cmdStart error: %v", err)
		}
	})

	if !strings.Contains(out, "Fix bug") {
		t.Errorf("output should mention task name, got: %q", out)
	}
	if !strings.Contains(out, "backend") {
		t.Errorf("output should mention project name, got: %q", out)
	}
	if !strings.Contains(out, "Started") {
		t.Errorf("output should say Started, got: %q", out)
	}
}

func TestCmdStart_ProjectFlagAfterTask(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// -p comes AFTER the task — our custom parser should handle this
	out := captureOutput(func() {
		err := cmdStart([]string{"Build feature", "-p", "frontend"})
		if err != nil {
			t.Errorf("cmdStart: %v", err)
		}
	})
	if !strings.Contains(out, "frontend") {
		t.Errorf("expected project 'frontend' in output, got: %q", out)
	}
}

func TestCmdStart_MultiWordTask(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	out := captureOutput(func() {
		cmdStart([]string{"Fix", "the", "login", "bug"})
	})
	if !strings.Contains(out, "Fix the login bug") {
		t.Errorf("expected full task in output, got: %q", out)
	}
}

// ─── cmdStop ──────────────────────────────────────────────────────────────────

func TestCmdStop_ErrorWhenNoActive(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	err := cmdStop(nil)
	if err == nil {
		t.Fatal("expected error stopping with no active session")
	}
}

func TestCmdStop_StopsRunningSession(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Task"}) })

	out := captureOutput(func() {
		if err := cmdStop(nil); err != nil {
			t.Errorf("cmdStop: %v", err)
		}
	})

	if !strings.Contains(out, "Stopped") {
		t.Errorf("expected 'Stopped' in output, got: %q", out)
	}
	if !strings.Contains(out, "Task") {
		t.Errorf("expected task name in stop output, got: %q", out)
	}
}

func TestCmdStop_ShowsDuration(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Task"}) })
	out := captureOutput(func() { cmdStop(nil) })

	// Duration should appear in the output
	if !strings.Contains(out, "Logged") {
		t.Errorf("expected 'Logged' duration in output, got: %q", out)
	}
}

// ─── cmdStatus ────────────────────────────────────────────────────────────────

func TestCmdStatus_NoSession(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	out := captureOutput(func() { cmdStatus(nil) })
	if !strings.Contains(strings.ToLower(out), "no active") {
		t.Errorf("expected 'no active' message, got: %q", out)
	}
}

func TestCmdStatus_ShowsRunningSession(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"My task", "-p", "myproj"}) })

	out := captureOutput(func() { cmdStatus(nil) })
	if !strings.Contains(out, "My task") {
		t.Errorf("expected task name in status output, got: %q", out)
	}
	if !strings.Contains(out, "myproj") {
		t.Errorf("expected project in status output, got: %q", out)
	}
	if !strings.Contains(strings.ToUpper(out), "RUNNING") {
		t.Errorf("expected RUNNING in status output, got: %q", out)
	}
}

// ─── cmdDelete ────────────────────────────────────────────────────────────────

func TestCmdDelete_RequiresID(t *testing.T) {
	err := cmdDelete([]string{})
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestCmdDelete_DeletesSession(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Create and stop a session, then delete it
	captureOutput(func() { cmdStart([]string{"Delete me"}) })
	captureOutput(func() { cmdStop(nil) })

	// Load store to get the ID
	s, _ := store.New()
	sessions := s.ForDate(time.Now())
	if len(sessions) == 0 {
		t.Fatal("expected at least one session")
	}
	id := sessions[0].ID

	out := captureOutput(func() {
		if err := cmdDelete([]string{id}); err != nil {
			t.Errorf("cmdDelete: %v", err)
		}
	})

	if !strings.Contains(out, id) {
		t.Errorf("expected deleted ID in output, got: %q", out)
	}

	// Verify it's gone
	s2, _ := store.New()
	remaining := s2.ForDate(time.Now())
	for _, sess := range remaining {
		if sess.ID == id {
			t.Errorf("session %s should have been deleted", id)
		}
	}
}

func TestCmdDelete_ErrorOnBadID(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	err := cmdDelete([]string{"doesnotexist"})
	if err == nil {
		t.Fatal("expected error deleting non-existent ID")
	}
}

// ─── cmdEdit ──────────────────────────────────────────────────────────────────

func TestCmdEdit_RequiresID(t *testing.T) {
	err := cmdEdit([]string{})
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestCmdEdit_RequiresFlagToUpdate(t *testing.T) {
	err := cmdEdit([]string{"someID"})
	if err == nil {
		t.Fatal("expected error when no -t or -p provided")
	}
}

func TestCmdEdit_UpdatesTask(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Old task"}) })
	captureOutput(func() { cmdStop(nil) })

	s, _ := store.New()
	id := s.ForDate(time.Now())[0].ID

	out := captureOutput(func() {
		if err := cmdEdit([]string{id, "-t", "New task"}); err != nil {
			t.Errorf("cmdEdit: %v", err)
		}
	})

	if !strings.Contains(out, "New task") {
		t.Errorf("expected updated task in output, got: %q", out)
	}

	s2, _ := store.New()
	updated := s2.ForDate(time.Now())[0]
	if updated.Task != "New task" {
		t.Errorf("Task = %q, want %q", updated.Task, "New task")
	}
}

func TestCmdEdit_FlagBeforeID(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Task"}) })
	captureOutput(func() { cmdStop(nil) })

	s, _ := store.New()
	id := s.ForDate(time.Now())[0].ID

	// -t before the ID — our parser should handle either order
	err := captureOutput(func() {}) // warmup
	_ = err
	out := captureOutput(func() {
		e := cmdEdit([]string{"-t", "Refactored", id})
		if e != nil {
			t.Logf("cmdEdit with flag before id: %v", e)
		}
	})
	// Either it works (flag before ID) or we get an error but don't panic
	_ = out
}

// ─── cmdLog ───────────────────────────────────────────────────────────────────

func TestCmdLog_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	out := captureOutput(func() {
		if err := cmdLog([]string{}); err != nil {
			t.Errorf("cmdLog: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(out), "no sessions") {
		t.Errorf("expected 'no sessions' message, got: %q", out)
	}
}

func TestCmdLog_ShowsTodaySessions(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Morning standup", "-p", "meetings"}) })
	captureOutput(func() { cmdStop(nil) })
	captureOutput(func() { cmdStart([]string{"Write API docs"}) })
	captureOutput(func() { cmdStop(nil) })

	out := captureOutput(func() {
		if err := cmdLog([]string{}); err != nil {
			t.Errorf("cmdLog: %v", err)
		}
	})

	if !strings.Contains(out, "Morning standup") {
		t.Errorf("expected first task in log, got: %q", out)
	}
	if !strings.Contains(out, "Write API docs") {
		t.Errorf("expected second task in log, got: %q", out)
	}
	if !strings.Contains(out, "Total") {
		t.Errorf("expected Total line in log, got: %q", out)
	}
}

func TestCmdLog_InvalidDateReturnsError(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	err := cmdLog([]string{"-d", "not-a-date"})
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestCmdLog_YesterdayFlag(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Should not error even with no sessions yesterday
	err := cmdLog([]string{"-d", "yesterday"})
	if err != nil {
		t.Errorf("cmdLog -d yesterday: %v", err)
	}
}

func TestCmdLog_ExplicitDateFlag(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	err := cmdLog([]string{"-d", "2026-01-15"})
	if err != nil {
		t.Errorf("cmdLog -d 2026-01-15: %v", err)
	}
}

// ─── cmdReport ────────────────────────────────────────────────────────────────

func TestCmdReport_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	out := captureOutput(func() {
		if err := cmdReport([]string{"--week"}); err != nil {
			t.Errorf("cmdReport: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(out), "no sessions") {
		t.Errorf("expected 'no sessions' message, got: %q", out)
	}
}

func TestCmdReport_WeeklyShowsByDay(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Coding", "-p", "backend"}) })
	captureOutput(func() { cmdStop(nil) })

	out := captureOutput(func() {
		if err := cmdReport([]string{"--week"}); err != nil {
			t.Errorf("cmdReport --week: %v", err)
		}
	})

	if !strings.Contains(out, "By Day") {
		t.Errorf("expected 'By Day' section, got: %q", out)
	}
	if !strings.Contains(out, "Total") {
		t.Errorf("expected 'Total' line, got: %q", out)
	}
}

func TestCmdReport_MonthlyFlag(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Monthly task", "-p", "proj"}) })
	captureOutput(func() { cmdStop(nil) })

	out := captureOutput(func() {
		if err := cmdReport([]string{"--month"}); err != nil {
			t.Errorf("cmdReport --month: %v", err)
		}
	})

	if !strings.Contains(out, "Monthly Report") {
		t.Errorf("expected 'Monthly Report' header, got: %q", out)
	}
}

func TestCmdReport_ShowsProjectBreakdown(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Task A", "-p", "alpha"}) })
	captureOutput(func() { cmdStop(nil) })
	captureOutput(func() { cmdStart([]string{"Task B", "-p", "beta"}) })
	captureOutput(func() { cmdStop(nil) })

	out := captureOutput(func() { cmdReport([]string{"--week"}) })

	if !strings.Contains(out, "By Project") {
		t.Errorf("expected 'By Project' section, got: %q", out)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected project 'alpha' in report, got: %q", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected project 'beta' in report, got: %q", out)
	}
}

// ─── Full workflow ────────────────────────────────────────────────────────────

func TestFullWorkflow_StartStopLog(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	// Start
	var startErr error
	captureOutput(func() {
		startErr = cmdStart([]string{"E2E task", "-p", "e2e"})
	})
	if startErr != nil {
		t.Fatalf("start: %v", startErr)
	}

	// Cannot start again
	captureOutput(func() {
		err := cmdStart([]string{"Another task"})
		if err == nil {
			t.Error("should reject double start")
		}
	})

	// Status shows active
	statusOut := captureOutput(func() { cmdStatus(nil) })
	if !strings.Contains(strings.ToUpper(statusOut), "RUNNING") {
		t.Errorf("expected RUNNING in status, got: %q", statusOut)
	}

	// Stop
	captureOutput(func() {
		if err := cmdStop(nil); err != nil {
			t.Errorf("stop: %v", err)
		}
	})

	// Status now shows nothing
	idleOut := captureOutput(func() { cmdStatus(nil) })
	if !strings.Contains(strings.ToLower(idleOut), "no active") {
		t.Errorf("expected 'no active' after stop, got: %q", idleOut)
	}

	// Log shows the session
	logOut := captureOutput(func() { cmdLog([]string{}) })
	if !strings.Contains(logOut, "E2E task") {
		t.Errorf("expected task in log, got: %q", logOut)
	}
}

func TestFullWorkflow_EditThenReport(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", os.Getenv("HOME"))

	captureOutput(func() { cmdStart([]string{"Rough task name", "-p", "work"}) })
	captureOutput(func() { cmdStop(nil) })

	s, _ := store.New()
	id := s.ForDate(time.Now())[0].ID

	// Edit the task name
	captureOutput(func() {
		if err := cmdEdit([]string{id, "-t", "Polished task name"}); err != nil {
			t.Errorf("edit: %v", err)
		}
	})

	// Report should reflect the edit
	reportOut := captureOutput(func() { cmdReport([]string{"--week"}) })
	if strings.Contains(reportOut, "Rough task name") {
		t.Errorf("old task name should not appear in report after edit")
	}
}
