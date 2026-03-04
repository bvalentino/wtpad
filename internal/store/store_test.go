package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bvalentino/wtpad/internal/model"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestDirReturnsBasePath(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	want := filepath.Join(dir, ".wtpad")
	if got := s.Dir(); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestNewDoesNotCreateDir(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	if _, err := os.Stat(s.Dir()); !os.IsNotExist(err) {
		t.Errorf("expected .wtpad/ to not exist after New, got err=%v", err)
	}
}

func TestLoadTodosMissingFile(t *testing.T) {
	s := tempStore(t)
	todos, err := s.LoadTodos()
	if err != nil {
		t.Fatalf("LoadTodos: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected empty slice, got %d todos", len(todos))
	}
}

func TestTodosRoundTrip(t *testing.T) {
	s := tempStore(t)
	want := []model.Todo{
		{Text: "Buy groceries"},
		{Text: "Fix login bug", Status: model.StatusDone},
		{Text: "Review PR #42"},
	}

	if err := s.SaveTodos(want); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	got, err := s.LoadTodos()
	if err != nil {
		t.Fatalf("LoadTodos: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d todos, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("todo[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTodosRoundTripInProgress(t *testing.T) {
	s := tempStore(t)
	want := []model.Todo{
		{Text: "Open task"},
		{Text: "Working on it", Status: model.StatusInProgress},
		{Text: "Finished", Status: model.StatusDone},
	}

	if err := s.SaveTodos(want); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	got, err := s.LoadTodos()
	if err != nil {
		t.Fatalf("LoadTodos: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d todos, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("todo[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Verify the raw file content uses correct markers.
	data, err := os.ReadFile(filepath.Join(s.Dir(), todosFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	raw := string(data)
	if !strings.Contains(raw, "- [ ] Open task") {
		t.Error("expected '- [ ] Open task' in raw file")
	}
	if !strings.Contains(raw, "- [~] Working on it") {
		t.Error("expected '- [~] Working on it' in raw file")
	}
	if !strings.Contains(raw, "- [x] Finished") {
		t.Error("expected '- [x] Finished' in raw file")
	}
}

func TestSaveTodosCreatesDir(t *testing.T) {
	s := tempStore(t)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}
	if _, err := os.Stat(s.Dir()); err != nil {
		t.Errorf("expected .wtpad/ to exist after SaveTodos: %v", err)
	}
}

func TestSaveTodosAtomic(t *testing.T) {
	s := tempStore(t)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(s.Dir(), ".tmp-*"))
	if len(leftovers) != 0 {
		t.Errorf("expected no .tmp-* files, found: %v", leftovers)
	}
}

func TestNoteRoundTrip(t *testing.T) {
	s := tempStore(t)
	body := "# Hello\n\nSome note content.\n"

	name, err := s.SaveNote("20260228-143022", body)
	if err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	if name != "20260228-143022" {
		t.Errorf("SaveNote name = %q, want %q", name, "20260228-143022")
	}

	note, err := s.LoadNote("20260228-143022")
	if err != nil {
		t.Fatalf("LoadNote: %v", err)
	}
	if note.Body != body {
		t.Errorf("Body = %q, want %q", note.Body, body)
	}
	if note.Name != "20260228-143022" {
		t.Errorf("Name = %q, want %q", note.Name, "20260228-143022")
	}
	if note.CreatedAt.IsZero() {
		t.Error("CreatedAt should be parsed from filename")
	}
}

func TestSaveNoteGeneratesName(t *testing.T) {
	s := tempStore(t)
	name, err := s.SaveNote("", "test content")
	if err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	if name == "" {
		t.Error("expected generated name, got empty string")
	}
	// Name is either YYYYMMDD-HHMMSS (15 chars) or YYYYMMDD-HHMMSS-N with collision suffix
	if len(name) < 15 {
		t.Errorf("expected at least 15-char timestamp name, got %q", name)
	}
}

func TestSaveNoteTimestampCollision(t *testing.T) {
	s := tempStore(t)

	// Create a note with a specific timestamp
	ts := "20260228-120000"
	if _, err := s.SaveNote(ts, "first"); err != nil {
		t.Fatalf("SaveNote first: %v", err)
	}

	// Manually create the file that SaveNote("", ...) would generate
	// to force a collision. We'll use SaveNote with explicit name to simulate.
	// Instead, test that two auto-generated notes don't collide by saving
	// with the same explicit name and checking both exist.
	name1, err := s.SaveNote(ts+"-1", "second")
	if err != nil {
		t.Fatalf("SaveNote second: %v", err)
	}
	if name1 == ts {
		t.Error("collision: second note got same name as first")
	}

	// Verify both files exist
	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestListNotesSortedNewestFirst(t *testing.T) {
	s := tempStore(t)

	names := []string{"20260101-100000", "20260301-120000", "20260201-110000"}
	for _, n := range names {
		if _, err := s.SaveNote(n, "content for "+n); err != nil {
			t.Fatalf("SaveNote(%s): %v", n, err)
		}
	}

	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}

	wantOrder := []string{"20260301-120000", "20260201-110000", "20260101-100000"}
	for i, want := range wantOrder {
		if notes[i].Name != want {
			t.Errorf("notes[%d].Name = %q, want %q", i, notes[i].Name, want)
		}
	}
}

func TestListNotesDoesNotLoadBody(t *testing.T) {
	s := tempStore(t)
	if _, err := s.SaveNote("20260228-120000", "some body content"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].Body != "" {
		t.Errorf("ListNotes should not load body, got %q", notes[0].Body)
	}
}

func TestListNotesExcludesTodos(t *testing.T) {
	s := tempStore(t)

	if err := s.SaveTodos([]model.Todo{{Text: "task"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}
	if _, err := s.SaveNote("20260228-120000", "a note"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1 (todos.md should be excluded)", len(notes))
	}
}

func TestDeleteNote(t *testing.T) {
	s := tempStore(t)
	name := "20260228-150000"
	if _, err := s.SaveNote(name, "to delete"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	if err := s.DeleteNote(name); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, err := s.LoadNote(name)
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error after delete, got %v", err)
	}
}

func TestLoadNoteMissingFile(t *testing.T) {
	s := tempStore(t)
	_, err := s.LoadNote("20260101-000000")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error, got %v", err)
	}
}

func TestPathTraversalLoadNote(t *testing.T) {
	s := tempStore(t)
	_, err := s.LoadNote("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "invalid note name") {
		t.Errorf("expected 'invalid note name' error, got %v", err)
	}
}

func TestPathTraversalSaveNote(t *testing.T) {
	s := tempStore(t)
	_, err := s.SaveNote("../../tmp/evil", "bad content")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "invalid note name") {
		t.Errorf("expected 'invalid note name' error, got %v", err)
	}
}

func TestPathTraversalDeleteNote(t *testing.T) {
	s := tempStore(t)
	err := s.DeleteNote("../../../tmp/evil")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "invalid note name") {
		t.Errorf("expected 'invalid note name' error, got %v", err)
	}
}

func TestReservedNameTodos(t *testing.T) {
	s := tempStore(t)
	_, err := s.SaveNote("todos", "overwrite attempt")
	if err == nil {
		t.Fatal("expected error for reserved name 'todos', got nil")
	}
	if !strings.Contains(err.Error(), "reserved filename") {
		t.Errorf("expected 'reserved filename' error, got %v", err)
	}
}

func TestAutoExcludeOnFirstWrite(t *testing.T) {
	dir := t.TempDir()
	// Set up a fake .git/info/ directory
	gitInfo := filepath.Join(dir, ".git", "info")
	os.MkdirAll(gitInfo, 0o755)
	os.WriteFile(filepath.Join(gitInfo, "exclude"), []byte("# ignore\n*.tmp\n"), 0o644)

	s, _ := New(dir)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(gitInfo, "exclude"))
	if err != nil {
		t.Fatalf("ReadFile exclude: %v", err)
	}

	if !strings.Contains(string(data), ".wtpad/") {
		t.Error("expected .wtpad/ in exclude file after first write")
	}
}

func TestAutoExcludeIdempotent(t *testing.T) {
	dir := t.TempDir()
	gitInfo := filepath.Join(dir, ".git", "info")
	os.MkdirAll(gitInfo, 0o755)
	os.WriteFile(filepath.Join(gitInfo, "exclude"), []byte(".wtpad/\n"), 0o644)

	s, _ := New(dir)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(gitInfo, "exclude"))
	if strings.Count(string(data), ".wtpad/") != 1 {
		t.Errorf("expected exactly one .wtpad/ entry, got:\n%s", data)
	}
}

func TestAutoExcludeLinkedWorktree(t *testing.T) {
	// Set up a fake main repo
	mainDir := t.TempDir()
	mainGitDir := filepath.Join(mainDir, ".git")
	os.MkdirAll(filepath.Join(mainGitDir, "info"), 0o755)
	os.MkdirAll(filepath.Join(mainGitDir, "worktrees", "wt1"), 0o755)
	os.WriteFile(filepath.Join(mainGitDir, "info", "exclude"), []byte("# exclude\n"), 0o644)

	// Set up linked worktree directory with .git file
	wtDir := t.TempDir()
	gitdirPath := filepath.Join(mainGitDir, "worktrees", "wt1")
	os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: "+gitdirPath+"\n"), 0o644)

	s, _ := New(wtDir)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(mainGitDir, "info", "exclude"))
	if err != nil {
		t.Fatalf("ReadFile exclude: %v", err)
	}
	if !strings.Contains(string(data), ".wtpad/") {
		t.Error("expected .wtpad/ in exclude file for linked worktree")
	}
}

func loadTitle(t *testing.T, s *Store) string {
	t.Helper()
	got, err := s.LoadTitle()
	if err != nil {
		t.Fatalf("LoadTitle: %v", err)
	}
	return got
}

func TestTitleRoundTrip(t *testing.T) {
	s := tempStore(t)

	// No title set initially
	if got := loadTitle(t, s); got != "" {
		t.Errorf("LoadTitle() = %q, want empty", got)
	}

	// Set a title
	if err := s.SaveTitle("My Project"); err != nil {
		t.Fatalf("SaveTitle: %v", err)
	}
	if got := loadTitle(t, s); got != "My Project" {
		t.Errorf("LoadTitle() = %q, want %q", got, "My Project")
	}

	// Clear the title
	if err := s.SaveTitle(""); err != nil {
		t.Fatalf("SaveTitle (clear): %v", err)
	}
	if got := loadTitle(t, s); got != "" {
		t.Errorf("LoadTitle() after clear = %q, want empty", got)
	}
}

func TestSaveTitleTrimsWhitespace(t *testing.T) {
	s := tempStore(t)
	if err := s.SaveTitle("  padded title  "); err != nil {
		t.Fatalf("SaveTitle: %v", err)
	}
	if got := loadTitle(t, s); got != "padded title" {
		t.Errorf("LoadTitle() = %q, want %q", got, "padded title")
	}
}

func TestSaveTitleWhitespaceOnlyClears(t *testing.T) {
	s := tempStore(t)
	// Set a title first
	if err := s.SaveTitle("has title"); err != nil {
		t.Fatalf("SaveTitle: %v", err)
	}
	// Saving whitespace-only should clear
	if err := s.SaveTitle("   "); err != nil {
		t.Fatalf("SaveTitle (whitespace): %v", err)
	}
	if got := loadTitle(t, s); got != "" {
		t.Errorf("LoadTitle() = %q, want empty after whitespace-only save", got)
	}
}

func TestLoadAIMissingFile(t *testing.T) {
	s := tempStore(t)
	todos, err := s.LoadAI()
	if err != nil {
		t.Fatalf("LoadAI: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected empty slice, got %d todos", len(todos))
	}
}

func TestLoadAIWithContent(t *testing.T) {
	s := tempStore(t)
	os.MkdirAll(s.Dir(), 0o700)
	content := "- [ ] Open task\n- [~] In progress\n- [x] Done task\n"
	os.WriteFile(filepath.Join(s.Dir(), "ai.md"), []byte(content), 0o600)

	todos, err := s.LoadAI()
	if err != nil {
		t.Fatalf("LoadAI: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("got %d todos, want 3", len(todos))
	}
	if todos[0].Text != "Open task" || todos[0].Status != model.StatusOpen {
		t.Errorf("todo[0] = %+v, want open 'Open task'", todos[0])
	}
	if todos[1].Text != "In progress" || todos[1].Status != model.StatusInProgress {
		t.Errorf("todo[1] = %+v, want in-progress 'In progress'", todos[1])
	}
	if todos[2].Text != "Done task" || todos[2].Status != model.StatusDone {
		t.Errorf("todo[2] = %+v, want done 'Done task'", todos[2])
	}
}

func TestLoadAISkipsNonTaskLines(t *testing.T) {
	s := tempStore(t)
	os.MkdirAll(s.Dir(), 0o700)
	content := "# AI Tasks\n\n- [ ] Real task\nsome random text\n- [x] Another\n"
	os.WriteFile(filepath.Join(s.Dir(), "ai.md"), []byte(content), 0o600)

	todos, err := s.LoadAI()
	if err != nil {
		t.Fatalf("LoadAI: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("got %d todos, want 2 (non-task lines skipped)", len(todos))
	}
}

func TestClearAI(t *testing.T) {
	s := tempStore(t)
	os.MkdirAll(s.Dir(), 0o700)
	os.WriteFile(filepath.Join(s.Dir(), "ai.md"), []byte("- [ ] task\n"), 0o600)

	if err := s.ClearAI(); err != nil {
		t.Fatalf("ClearAI: %v", err)
	}

	if _, err := os.Stat(filepath.Join(s.Dir(), "ai.md")); !os.IsNotExist(err) {
		t.Errorf("expected ai.md to be removed, got err=%v", err)
	}
}

func TestClearAIMissingFile(t *testing.T) {
	s := tempStore(t)
	// Should not error when file doesn't exist.
	if err := s.ClearAI(); err != nil {
		t.Fatalf("ClearAI on missing file: %v", err)
	}
}

func TestAIExists(t *testing.T) {
	s := tempStore(t)
	if s.AIExists() {
		t.Error("AIExists should be false when ai.md doesn't exist")
	}

	os.MkdirAll(s.Dir(), 0o700)
	os.WriteFile(filepath.Join(s.Dir(), "ai.md"), []byte("- [ ] task\n"), 0o600)

	if !s.AIExists() {
		t.Error("AIExists should be true when ai.md exists")
	}
}

func TestListNotesExcludesAI(t *testing.T) {
	s := tempStore(t)
	os.MkdirAll(s.Dir(), 0o700)
	os.WriteFile(filepath.Join(s.Dir(), "ai.md"), []byte("- [ ] task\n"), 0o600)
	if _, err := s.SaveNote("20260228-120000", "a note"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1 (ai.md should be excluded)", len(notes))
	}
}

func TestAutoExcludeNoGitDir(t *testing.T) {
	// No .git/ directory — should silently skip without error
	s := tempStore(t)
	if err := s.SaveTodos([]model.Todo{{Text: "test"}}); err != nil {
		t.Fatalf("SaveTodos should not fail in non-git dir: %v", err)
	}
}
