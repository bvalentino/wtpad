package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bvalentino/wtpad/internal/model"
)

const (
	todosFile   = "todos.md"
	noteTimeFmt = "20060102-150405"
)

// Store handles all disk I/O for the .wtpad/ directory.
type Store struct {
	basePath string
}

// New creates a Store rooted at dir/.wtpad/.
// It does not create the directory; that happens lazily on first write.
func New(dir string) (*Store, error) {
	return &Store{
		basePath: filepath.Join(dir, ".wtpad"),
	}, nil
}

// Dir returns the .wtpad/ directory path.
func (s *Store) Dir() string {
	return s.basePath
}

// ensureDir creates the .wtpad/ directory if it doesn't exist.
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.basePath, 0o755)
}

// validNoteName checks that name does not escape basePath or collide with reserved files.
func (s *Store) validNoteName(name string) error {
	if name == "" {
		return fmt.Errorf("empty note name")
	}
	p := filepath.Join(s.basePath, name+".md")
	cleaned := filepath.Clean(p)
	if !strings.HasPrefix(cleaned, filepath.Clean(s.basePath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid note name: %q", name)
	}
	if filepath.Base(cleaned) == todosFile {
		return fmt.Errorf("reserved filename: %q", name)
	}
	return nil
}

// LoadTodos reads todos.md and parses GFM task list lines.
// Returns an empty slice if the file does not exist.
func (s *Store) LoadTodos() ([]model.Todo, error) {
	path := filepath.Join(s.basePath, todosFile)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []model.Todo{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var todos []model.Todo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if todo, ok := parseTodoLine(line); ok {
			todos = append(todos, todo)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

// parseTodoLine parses a single GFM task list line.
func parseTodoLine(line string) (model.Todo, bool) {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "- [x] "):
		return model.Todo{Text: line[6:], Done: true}, true
	case strings.HasPrefix(line, "- [ ] "):
		return model.Todo{Text: line[6:], Done: false}, true
	default:
		return model.Todo{}, false
	}
}

// SaveTodos writes todos as a GFM task list to todos.md atomically.
func (s *Store) SaveTodos(todos []model.Todo) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	var buf strings.Builder
	for _, t := range todos {
		if t.Done {
			fmt.Fprintf(&buf, "- [x] %s\n", t.Text)
		} else {
			fmt.Fprintf(&buf, "- [ ] %s\n", t.Text)
		}
	}

	return s.atomicWrite(todosFile, buf.String())
}

// ListNotes scans .wtpad/ for note files, excluding todos.md.
// Returns notes sorted by filename descending (newest first).
// Only reads metadata (name + timestamp) — use LoadNote to get the body.
func (s *Store) ListNotes() ([]model.Note, error) {
	entries, err := filepath.Glob(filepath.Join(s.basePath, "*.md"))
	if err != nil {
		return nil, err
	}

	var notes []model.Note
	for _, path := range entries {
		base := filepath.Base(path)
		if base == todosFile {
			continue
		}
		name := strings.TrimSuffix(base, ".md")
		// Non-timestamp names get zero time
		createdAt, _ := time.Parse(noteTimeFmt, name)
		notes = append(notes, model.Note{
			Name:      name,
			CreatedAt: createdAt,
		})
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Name > notes[j].Name
	})

	return notes, nil
}

// LoadNote reads a single note file by name (without .md extension).
func (s *Store) LoadNote(name string) (*model.Note, error) {
	if err := s.validNoteName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(s.basePath, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Non-timestamp names get zero time
	createdAt, _ := time.Parse(noteTimeFmt, name)

	return &model.Note{
		Name:      name,
		Body:      string(data),
		CreatedAt: createdAt,
	}, nil
}

// SaveNote writes a note atomically. If name is empty, generates a timestamp name.
// Returns the name used.
func (s *Store) SaveNote(name string, body string) (string, error) {
	if err := s.ensureDir(); err != nil {
		return "", err
	}

	if name == "" {
		name = time.Now().Format(noteTimeFmt)
		// Avoid collision: append incrementing suffix if file exists
		if _, err := os.Stat(filepath.Join(s.basePath, name+".md")); err == nil {
			for i := 1; i < 100; i++ {
				candidate := fmt.Sprintf("%s-%d", name, i)
				if _, err := os.Stat(filepath.Join(s.basePath, candidate+".md")); os.IsNotExist(err) {
					name = candidate
					break
				}
			}
		}
	}

	if err := s.validNoteName(name); err != nil {
		return "", err
	}

	if err := s.atomicWrite(name+".md", body); err != nil {
		return "", err
	}
	return name, nil
}

// DeleteNote removes a note file.
func (s *Store) DeleteNote(name string) error {
	if err := s.validNoteName(name); err != nil {
		return err
	}
	path := filepath.Join(s.basePath, name+".md")
	return os.Remove(path)
}

// atomicWrite writes data to a .tmp file then renames it into place.
func (s *Store) atomicWrite(filename, data string) error {
	target := filepath.Join(s.basePath, filename)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return err
	}
	return nil
}
