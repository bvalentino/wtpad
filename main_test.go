package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	cmd := extractHookCommand(t, settings)
	if !strings.Contains(cmd, "wtpad ai prompt") {
		t.Errorf("hook command missing 'wtpad ai prompt': %s", cmd)
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

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	tools, ok := settings["allowedTools"].([]any)
	if !ok || len(tools) != 2 {
		t.Error("existing allowedTools was not preserved")
	}
	if extractHookCommand(t, settings) == "" {
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

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	cmd := extractHookCommand(t, settings)
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

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	cmd := extractHookCommand(t, settings)
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

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

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

func TestRemoveSettingsHook_RemovesWtpadHook(t *testing.T) {
	dir := t.TempDir()
	// Install first
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if !removed {
		t.Error("expected hook to be removed")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	if extractHookCommand(t, settings) != "" {
		t.Error("wtpad hook still present after removal")
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

	// Install wtpad hook alongside existing
	if err := mergeSettingsHook(dir); err != nil {
		t.Fatalf("mergeSettingsHook: %v", err)
	}

	// Remove wtpad hook
	removed, err := removeSettingsHook(dir)
	if err != nil {
		t.Fatalf("removeSettingsHook: %v", err)
	}
	if !removed {
		t.Error("expected hook to be removed")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	sessionStart := hooks["SessionStart"].([]any)
	if len(sessionStart) != 1 {
		t.Errorf("expected 1 remaining SessionStart entry, got %d", len(sessionStart))
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

// extractHookCommand finds the wtpad hook command in the settings structure.
func extractHookCommand(t *testing.T, settings map[string]any) string {
	t.Helper()
	hooks, _ := settings["hooks"].(map[string]any)
	sessionStart, _ := hooks["SessionStart"].([]any)
	for _, entry := range sessionStart {
		em, _ := entry.(map[string]any)
		// Check nested format
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
