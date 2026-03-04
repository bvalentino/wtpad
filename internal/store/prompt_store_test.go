package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempPromptStore(t *testing.T) *PromptStore {
	t.Helper()
	return NewPromptStore(filepath.Join(t.TempDir(), "prompts"))
}

func TestListPromptsEmptyDir(t *testing.T) {
	ps := tempPromptStore(t)
	prompts, err := ps.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(prompts) != 0 {
		t.Errorf("expected empty slice, got %d prompts", len(prompts))
	}
}

func TestPromptRoundTrip(t *testing.T) {
	ps := tempPromptStore(t)

	name, err := ps.SavePrompt("my-prompt", "Hello world")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}
	if name != "my-prompt" {
		t.Errorf("SavePrompt name = %q, want %q", name, "my-prompt")
	}

	got, err := ps.LoadPrompt("my-prompt")
	if err != nil {
		t.Fatalf("LoadPrompt: %v", err)
	}
	if got.Name != "my-prompt" {
		t.Errorf("Name = %q, want %q", got.Name, "my-prompt")
	}
	if got.Body != "Hello world" {
		t.Errorf("Body = %q, want %q", got.Body, "Hello world")
	}
}

func TestPromptAutoName(t *testing.T) {
	ps := tempPromptStore(t)

	name, err := ps.SavePrompt("", "auto-named prompt")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}
	if name == "" {
		t.Fatal("expected auto-generated name, got empty string")
	}

	got, err := ps.LoadPrompt(name)
	if err != nil {
		t.Fatalf("LoadPrompt: %v", err)
	}
	if got.Body != "auto-named prompt" {
		t.Errorf("Body = %q, want %q", got.Body, "auto-named prompt")
	}
}

func TestListPromptsSortedNewestFirst(t *testing.T) {
	ps := tempPromptStore(t)
	ps.SavePrompt("20260101-100000", "older")
	ps.SavePrompt("20260201-100000", "newer")

	prompts, err := ps.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("got %d prompts, want 2", len(prompts))
	}
	if prompts[0].Name != "20260201-100000" {
		t.Errorf("prompts[0].Name = %q, want newest first", prompts[0].Name)
	}
}

func TestDeletePrompt(t *testing.T) {
	ps := tempPromptStore(t)
	ps.SavePrompt("todelete", "bye")

	if err := ps.DeletePrompt("todelete"); err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}

	_, err := ps.LoadPrompt("todelete")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error after delete, got %v", err)
	}
}

func TestPromptPathTraversal(t *testing.T) {
	ps := tempPromptStore(t)

	cases := []struct {
		name string
		op   string
	}{
		{"../../etc/passwd", "load"},
		{"../evil", "save"},
		{"../../tmp/hack", "delete"},
	}

	for _, tc := range cases {
		t.Run(tc.op+"_"+tc.name, func(t *testing.T) {
			var err error
			switch tc.op {
			case "load":
				_, err = ps.LoadPrompt(tc.name)
			case "save":
				_, err = ps.SavePrompt(tc.name, "bad")
			case "delete":
				err = ps.DeletePrompt(tc.name)
			}
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
			if !strings.Contains(err.Error(), "invalid prompt name") {
				t.Errorf("expected 'invalid prompt name' error, got %v", err)
			}
		})
	}
}

func TestPromptEmptyName(t *testing.T) {
	ps := tempPromptStore(t)
	_, err := ps.LoadPrompt("")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !strings.Contains(err.Error(), "empty prompt name") {
		t.Errorf("expected 'empty prompt name' error, got %v", err)
	}
}

func TestSavePromptCreatesDirLazily(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "prompts")
	ps := NewPromptStore(dir)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("expected prompts dir to not exist before save")
	}

	if _, err := ps.SavePrompt("test", "content"); err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected prompts dir to exist after save: %v", err)
	}
}

func TestSavePromptAtomicCleanup(t *testing.T) {
	ps := tempPromptStore(t)
	if _, err := ps.SavePrompt("atomic", "content"); err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}
	tmp := filepath.Join(ps.basePath, "atomic.md.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("expected .tmp file to be cleaned up, got err=%v", err)
	}
}
