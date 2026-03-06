package model

import "time"

// TodoStatus represents the state of a todo item.
type TodoStatus int

const (
	StatusOpen       TodoStatus = iota
	StatusInProgress
	StatusDone
)

// Todo represents a single task list item, parsed from a GFM task list line.
type Todo struct {
	Text   string
	Status TodoStatus
}

// GFMLine returns the todo as a GFM task list line (e.g. "- [x] task text").
func (t Todo) GFMLine() string {
	switch t.Status {
	case StatusDone:
		return "- [x] " + t.Text
	case StatusInProgress:
		return "- [~] " + t.Text
	default:
		return "- [ ] " + t.Text
	}
}

// Note represents a single markdown note file in .wtpad/.
type Note struct {
	Name      string    // filename without .md extension (e.g., "20260228-143022")
	Body      string    // full markdown content
	CreatedAt time.Time // parsed from the filename timestamp
}

