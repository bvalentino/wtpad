package tui

import (
	"testing"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func tempStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func TestViewBeforeWindowSizeMsg(t *testing.T) {
	s := tempStore(t)
	app := New(s, []model.Todo{{Text: "task"}}, nil, "main")

	// View() is called before any WindowSizeMsg, so width/height are 0.
	// This must not panic — returns empty string gracefully.
	out := app.View()
	if out != "" {
		t.Errorf("expected empty output before WindowSizeMsg, got %q", out)
	}
}
