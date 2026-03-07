package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

// --- hasInProgress tests ---

func TestHasInProgress_Empty(t *testing.T) {
	if hasInProgress(nil) {
		t.Error("expected false for nil slice")
	}
	if hasInProgress([]model.Todo{}) {
		t.Error("expected false for empty slice")
	}
}

func TestHasInProgress_NoInProgress(t *testing.T) {
	todos := []model.Todo{
		{Text: "a", Status: model.StatusOpen},
		{Text: "b", Status: model.StatusDone},
	}
	if hasInProgress(todos) {
		t.Error("expected false when no in-progress tasks")
	}
}

func TestHasInProgress_WithInProgress(t *testing.T) {
	todos := []model.Todo{
		{Text: "a", Status: model.StatusOpen},
		{Text: "b", Status: model.StatusInProgress},
	}
	if !hasInProgress(todos) {
		t.Error("expected true when in-progress task exists")
	}
}

// --- cmdAIStart tests ---

func TestAIStart_TransitionsExistingOpenTask(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "my task", Status: model.StatusOpen}})
	captureStdout(t, func() { cmdAIStart(s, []string{"my task"}) })
	todos, _ := s.LoadAI()
	if len(todos) != 1 {
		t.Fatalf("expected 1 task, got %d", len(todos))
	}
	if todos[0].Status != model.StatusInProgress {
		t.Errorf("expected StatusInProgress, got %v", todos[0].Status)
	}
}

func TestAIStart_TransitionIsCaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "My Task", Status: model.StatusOpen}})
	captureStdout(t, func() { cmdAIStart(s, []string{"my", "task"}) })
	todos, _ := s.LoadAI()
	if len(todos) != 1 {
		t.Fatalf("expected 1 task, got %d", len(todos))
	}
	if todos[0].Status != model.StatusInProgress {
		t.Errorf("expected StatusInProgress, got %v", todos[0].Status)
	}
}

func TestAIStart_CreatesNewWhenNoMatch(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "other task", Status: model.StatusOpen}})
	captureStdout(t, func() { cmdAIStart(s, []string{"new task"}) })
	todos, _ := s.LoadAI()
	if len(todos) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(todos))
	}
	if todos[1].Text != "new task" || todos[1].Status != model.StatusInProgress {
		t.Errorf("expected new in-progress task, got %+v", todos[1])
	}
}

func TestAIStart_DoesNotTransitionDoneTask(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "done task", Status: model.StatusDone}})
	captureStdout(t, func() { cmdAIStart(s, []string{"done task"}) })
	todos, _ := s.LoadAI()
	if len(todos) != 2 {
		t.Fatalf("expected 2 tasks (done + new), got %d", len(todos))
	}
	if todos[0].Status != model.StatusDone {
		t.Error("original done task should remain done")
	}
	if todos[1].Status != model.StatusInProgress {
		t.Error("new task should be in-progress")
	}
}

// --- remind-start / remind-done tests ---

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func TestRemindStart_PrintsWhenNoTasks(t *testing.T) {
	s := newTestStore(t)
	out := captureStdout(t, func() { cmdAIRemindStart(s) })
	if !strings.Contains(out, "wtpad ai add") {
		t.Errorf("expected add reminder, got: %q", out)
	}
	if !strings.Contains(out, "wtpad ai start") {
		t.Errorf("expected start reminder, got: %q", out)
	}
	if !strings.Contains(out, "wtpad ai title") {
		t.Errorf("expected title reminder when no tasks, got: %q", out)
	}
}

func TestRemindStart_PrintsWhenOnlyOpenTasks(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "queued", Status: model.StatusOpen}})
	out := captureStdout(t, func() { cmdAIRemindStart(s) })
	if !strings.Contains(out, "wtpad ai start") {
		t.Errorf("expected start reminder, got: %q", out)
	}
	// Should NOT remind about title when tasks exist
	if strings.Contains(out, "wtpad ai title") {
		t.Error("should not remind about title when tasks already exist")
	}
}

func TestRemindStart_SilentWhenInProgress(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "working", Status: model.StatusInProgress}})
	out := captureStdout(t, func() { cmdAIRemindStart(s) })
	if out != "" {
		t.Errorf("expected silence when in-progress task exists, got: %q", out)
	}
}

func TestRemindStart_NoTitleReminderWhenTitleSet(t *testing.T) {
	s := newTestStore(t)
	s.SaveTitle("my session")
	out := captureStdout(t, func() { cmdAIRemindStart(s) })
	if strings.Contains(out, "wtpad ai title") {
		t.Error("should not remind about title when title already set")
	}
}

func TestRemindDone_PrintsWhenInProgress(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "working", Status: model.StatusInProgress}})
	out := captureStdout(t, func() { cmdAIRemindDone(s) })
	if !strings.Contains(out, "wtpad ai done") {
		t.Errorf("expected done reminder, got: %q", out)
	}
}

func TestRemindDone_SilentWhenNoTasks(t *testing.T) {
	s := newTestStore(t)
	out := captureStdout(t, func() { cmdAIRemindDone(s) })
	if out != "" {
		t.Errorf("expected silence when no tasks, got: %q", out)
	}
}

func TestRemindDone_SilentWhenAllDone(t *testing.T) {
	s := newTestStore(t)
	s.SaveAI([]model.Todo{{Text: "finished", Status: model.StatusDone}})
	out := captureStdout(t, func() { cmdAIRemindDone(s) })
	if out != "" {
		t.Errorf("expected silence when all done, got: %q", out)
	}
}

// captureStdout captures stdout from fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	buf, _ := io.ReadAll(r)
	return string(buf)
}

// --- mergeSettingsHook tests ---

func TestMergeSettingsHook_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	cmd := extractHookCommand(t, settings, "SessionStart")
	if !strings.Contains(cmd, "wtpad ai prompt") {
		t.Errorf("hook command missing 'wtpad ai prompt': %s", cmd)
	}
}

func TestMergeSettingsHook_InstallsAllThreeHooks(t *testing.T) {
	dir := t.TempDir()
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	settings := readSettings(t, dir)

	for _, tc := range []struct {
		event, subcmd string
	}{
		{"SessionStart", "wtpad ai prompt"},
		{"UserPromptSubmit", "wtpad ai remind-start"},
		{"Stop", "wtpad ai remind-done"},
	} {
		cmd := extractHookCommand(t, settings, tc.event)
		if !strings.Contains(cmd, tc.subcmd) {
			t.Errorf("%s hook missing %q, got: %s", tc.event, tc.subcmd, cmd)
		}
	}
}

func TestMergeSettingsHook_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "settings.json"))

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "settings.json"))

	if string(first) != string(second) {
		t.Errorf("mergeSettingsHook is not idempotent\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestMergeSettingsHook_PreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	existing := `{"allowedTools": ["Read", "Write"]}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644)

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	settings := readSettings(t, dir)

	tools, ok := settings["allowedTools"].([]any)
	if !ok || len(tools) != 2 {
		t.Error("existing allowedTools was not preserved")
	}
	if extractHookCommand(t, settings, "SessionStart") == "" {
		t.Error("hook was not added")
	}
}

func TestMergeSettingsHook_MigratesOldFlatFormat(t *testing.T) {
	dir := t.TempDir()
	oldSettings := `{
  "hooks": {
    "SessionStart": [
      {"type": "command", "command": "wtpad ai ls 2>/dev/null || true"}
    ]
  }
}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(oldSettings), 0o644)

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	settings := readSettings(t, dir)

	cmd := extractHookCommand(t, settings, "SessionStart")
	if !strings.Contains(cmd, "wtpad ai prompt") {
		t.Errorf("old hook was not migrated, got: %s", cmd)
	}

	// Should not have duplicated the entry
	hooks := settings["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	if len(sessionStart) != 1 {
		t.Errorf("expected 1 SessionStart entry after migration, got %d", len(sessionStart))
	}
}

func TestMergeSettingsHook_MigratesOldNestedFormat(t *testing.T) {
	dir := t.TempDir()
	oldSettings := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {"type": "command", "command": "wtpad ai ls 2>/dev/null || true"}
        ]
      }
    ]
  }
}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(oldSettings), 0o644)

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	settings := readSettings(t, dir)

	cmd := extractHookCommand(t, settings, "SessionStart")
	if !strings.Contains(cmd, "command -v wtpad") {
		t.Errorf("old nested hook was not migrated, got: %s", cmd)
	}

	hooks := settings["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	if len(sessionStart) != 1 {
		t.Errorf("expected 1 SessionStart entry after migration, got %d", len(sessionStart))
	}
}

func TestMergeSettingsHook_PreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	existing := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {"type": "command", "command": "echo hello"}
        ]
      }
    ]
  }
}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644)

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	settings := readSettings(t, dir)

	hooks := settings["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	if len(sessionStart) != 2 {
		t.Errorf("expected 2 SessionStart entries (existing + wtpad), got %d", len(sessionStart))
	}
}

func TestMergeSettingsHook_NoHTMLEscaping(t *testing.T) {
	dir := t.TempDir()
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	content := string(data)

	if strings.Contains(content, `\u003c`) || strings.Contains(content, `\u0026`) {
		t.Error("settings.json contains HTML-escaped characters")
	}
}

// --- removeSettingsHook tests ---

func TestRemoveSettingsHook_RemovesAllHookEvents(t *testing.T) {
	dir := t.TempDir()
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if !removed {
		t.Error("expected hooks to be removed")
	}

	settings := readSettings(t, dir)

	for _, event := range []string{"SessionStart", "UserPromptSubmit", "Stop"} {
		if extractHookCommand(t, settings, event) != "" {
			t.Errorf("wtpad hook still present in %s after removal", event)
		}
	}
}

func TestRemoveSettingsHook_PreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	existing := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {"type": "command", "command": "echo hello"}
        ]
      }
    ]
  }
}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644)

	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if !removed {
		t.Error("expected hooks to be removed")
	}

	settings := readSettings(t, dir)

	hooks := settings["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	if len(sessionStart) != 1 {
		t.Errorf("expected 1 remaining SessionStart entry, got %d", len(sessionStart))
	}
	// UserPromptSubmit and Stop should be fully cleaned up
	if _, exists := hooks["UserPromptSubmit"]; exists {
		t.Error("UserPromptSubmit should be removed when only wtpad hooks existed")
	}
	if _, exists := hooks["Stop"]; exists {
		t.Error("Stop should be removed when only wtpad hooks existed")
	}
}

func TestRemoveSettingsHook_NoHookPresent(t *testing.T) {
	dir := t.TempDir()
	existing := `{"allowedTools": ["Read"]}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644)

	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if removed {
		t.Error("expected no hook to be removed")
	}
}

func TestRemoveSettingsHook_NoFile(t *testing.T) {
	dir := t.TempDir()
	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if removed {
		t.Error("expected no hook to be removed when file doesn't exist")
	}
}

func TestRemoveSettingsHook_CleansEmptyHooksKey(t *testing.T) {
	dir := t.TempDir()
	// Install only wtpad hook
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	removeSettingsHook(dir)

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	if _, exists := settings["hooks"]; exists {
		t.Error("expected empty hooks key to be removed")
	}
}

// readSettings reads and parses settings.json from a test directory.
func readSettings(t *testing.T, dir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return settings
}

// extractHookCommand finds the wtpad hook command for a given event in the settings structure.
func extractHookCommand(t *testing.T, settings map[string]any, event string) string {
	t.Helper()
	hooks, _ := settings["hooks"].(map[string]any)
	entries, _ := hooks[event].([]any)
	for _, entry := range entries {
		em, _ := entry.(map[string]any)
		hooksArr, _ := em["hooks"].([]any)
		for _, h := range hooksArr {
			hm, _ := h.(map[string]any)
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, "wtpad") {
				return cmd
			}
		}
	}
	return ""
}
