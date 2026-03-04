package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bvalentino/wtpad/internal/model"
)

func tempTemplateStore(t *testing.T) *TemplateStore {
	t.Helper()
	return NewTemplateStore(filepath.Join(t.TempDir(), "templates"))
}

func TestListTemplatesEmptyDir(t *testing.T) {
	ts := tempTemplateStore(t)
	templates, err := ts.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected empty slice, got %d templates", len(templates))
	}
}

func TestTemplateRoundTrip(t *testing.T) {
	ts := tempTemplateStore(t)
	todos := []model.Todo{
		{Text: "Step 1"},
		{Text: "Step 2"},
		{Text: "Step 3", Status: model.StatusDone},
	}

	name, err := ts.SaveTemplate("workflow", todos)
	if err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}
	if name != "workflow" {
		t.Errorf("SaveTemplate name = %q, want %q", name, "workflow")
	}

	got, err := ts.LoadTemplate("workflow")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	if len(got) != len(todos) {
		t.Fatalf("got %d todos, want %d", len(got), len(todos))
	}
	for i := range todos {
		if got[i] != todos[i] {
			t.Errorf("todo[%d] = %+v, want %+v", i, got[i], todos[i])
		}
	}
}

func TestListTemplatesReturnsCountsAndSorted(t *testing.T) {
	ts := tempTemplateStore(t)
	ts.SaveTemplate("beta", []model.Todo{{Text: "one"}, {Text: "two"}})
	ts.SaveTemplate("alpha", []model.Todo{{Text: "single"}})

	templates, err := ts.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("got %d templates, want 2", len(templates))
	}
	// Sorted alphabetically
	if templates[0].Name != "alpha" {
		t.Errorf("templates[0].Name = %q, want %q", templates[0].Name, "alpha")
	}
	if templates[0].TodoCount != 1 {
		t.Errorf("templates[0].TodoCount = %d, want 1", templates[0].TodoCount)
	}
	if templates[1].Name != "beta" {
		t.Errorf("templates[1].Name = %q, want %q", templates[1].Name, "beta")
	}
	if templates[1].TodoCount != 2 {
		t.Errorf("templates[1].TodoCount = %d, want 2", templates[1].TodoCount)
	}
}

func TestTemplateOverwrite(t *testing.T) {
	ts := tempTemplateStore(t)
	ts.SaveTemplate("myflow", []model.Todo{{Text: "old"}})
	ts.SaveTemplate("myflow", []model.Todo{{Text: "new1"}, {Text: "new2"}})

	got, err := ts.LoadTemplate("myflow")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d todos, want 2", len(got))
	}
	if got[0].Text != "new1" {
		t.Errorf("todo[0].Text = %q, want %q", got[0].Text, "new1")
	}
}

func TestTemplatePathTraversal(t *testing.T) {
	ts := tempTemplateStore(t)

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
				_, err = ts.LoadTemplate(tc.name)
			case "save":
				_, err = ts.SaveTemplate(tc.name, []model.Todo{{Text: "x"}})
			case "delete":
				err = ts.DeleteTemplate(tc.name)
			}
			if err == nil {
				t.Fatal("expected error for path traversal, got nil")
			}
			if !strings.Contains(err.Error(), "invalid template name") {
				t.Errorf("expected 'invalid template name' error, got %v", err)
			}
		})
	}
}

func TestTemplateEmptyName(t *testing.T) {
	ts := tempTemplateStore(t)
	_, err := ts.SaveTemplate("", []model.Todo{{Text: "x"}})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !strings.Contains(err.Error(), "empty template name") {
		t.Errorf("expected 'empty template name' error, got %v", err)
	}
}

func TestDeleteTemplate(t *testing.T) {
	ts := tempTemplateStore(t)
	ts.SaveTemplate("todelete", []model.Todo{{Text: "x"}})

	if err := ts.DeleteTemplate("todelete"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	_, err := ts.LoadTemplate("todelete")
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error after delete, got %v", err)
	}
}

func TestTemplateExistsCheck(t *testing.T) {
	ts := tempTemplateStore(t)
	if ts.TemplateExists("nope") {
		t.Error("TemplateExists should return false for missing template")
	}

	ts.SaveTemplate("exists", []model.Todo{{Text: "x"}})
	if !ts.TemplateExists("exists") {
		t.Error("TemplateExists should return true after save")
	}
}

func TestSaveTemplateCreatesDirLazily(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")
	ts := NewTemplateStore(dir)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("expected templates dir to not exist before save")
	}

	if _, err := ts.SaveTemplate("test", []model.Todo{{Text: "x"}}); err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected templates dir to exist after save: %v", err)
	}
}

func TestSaveTemplateAtomicCleanup(t *testing.T) {
	ts := tempTemplateStore(t)
	if _, err := ts.SaveTemplate("atomic", []model.Todo{{Text: "x"}}); err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(ts.basePath, ".tmp-*"))
	if len(leftovers) != 0 {
		t.Errorf("expected no .tmp-* files, found: %v", leftovers)
	}
}
