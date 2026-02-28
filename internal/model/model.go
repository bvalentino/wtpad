package model

import "time"

// Todo represents a single task list item, parsed from a GFM task list line.
type Todo struct {
	Text string
	Done bool
}

// Note represents a single markdown note file in .wtpad/.
type Note struct {
	Name      string    // filename without .md extension (e.g., "20260228-143022")
	Body      string    // full markdown content
	CreatedAt time.Time // parsed from the filename timestamp
}
