package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gitutil "github.com/bvalentino/wtpad/internal/git"
	"github.com/bvalentino/wtpad/internal/model"
)

const (
	todosFile   = "todos.md"
	aiFile      = "ai.md"
	titleFile   = "title.txt"
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
// On first creation, it adds .wtpad/ to .git/info/exclude.
func (s *Store) ensureDir() error {
	_, statErr := os.Stat(s.basePath)
	created := os.IsNotExist(statErr)

	if err := os.MkdirAll(s.basePath, 0o700); err != nil {
		return err
	}

	if created {
		s.autoExclude()
	}
	return nil
}

// autoExclude appends .wtpad/ to .git/info/exclude if not already present.
// Silently skips if not in a git repo or on any error.
func (s *Store) autoExclude() {
	gitDir := gitutil.FindGitDir(filepath.Dir(s.basePath))
	if gitDir == "" {
		return
	}
	s.appendExclude(filepath.Join(gitDir, "info", "exclude"))
}

func (s *Store) appendExclude(excludePath string) {
	// Ensure the info/ directory exists
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return
	}

	data, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == ".wtpad/" {
			return // already excluded
		}
	}

	f, err := os.OpenFile(excludePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	// Add newline before entry if file doesn't end with one
	prefix := ""
	if len(data) > 0 && data[len(data)-1] != '\n' {
		prefix = "\n"
	}
	fmt.Fprintf(f, "%s.wtpad/\n", prefix)
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

// loadTodoFile reads a GFM task list file and returns parsed todos.
// Returns an empty slice if the file does not exist.
func (s *Store) loadTodoFile(filename string) ([]model.Todo, error) {
	path := filepath.Join(s.basePath, filename)
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

// LoadTodos reads todos.md and parses GFM task list lines.
func (s *Store) LoadTodos() ([]model.Todo, error) {
	return s.loadTodoFile(todosFile)
}

// parseTodoLine parses a single GFM task list line.
func parseTodoLine(line string) (model.Todo, bool) {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "- [x] "):
		return model.Todo{Text: line[6:], Status: model.StatusDone}, true
	case strings.HasPrefix(line, "- [~] "):
		return model.Todo{Text: line[6:], Status: model.StatusInProgress}, true
	case strings.HasPrefix(line, "- [ ] "):
		return model.Todo{Text: line[6:], Status: model.StatusOpen}, true
	default:
		return model.Todo{}, false
	}
}

// saveTodoFile writes todos as a GFM task list to the given file atomically.
func (s *Store) saveTodoFile(filename string, todos []model.Todo) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	var buf strings.Builder
	for _, t := range todos {
		switch t.Status {
		case model.StatusDone:
			fmt.Fprintf(&buf, "- [x] %s\n", t.Text)
		case model.StatusInProgress:
			fmt.Fprintf(&buf, "- [~] %s\n", t.Text)
		default:
			fmt.Fprintf(&buf, "- [ ] %s\n", t.Text)
		}
	}
	return s.atomicWrite(filename, buf.String())
}

// SaveTodos writes todos as a GFM task list to todos.md atomically.
func (s *Store) SaveTodos(todos []model.Todo) error {
	return s.saveTodoFile(todosFile, todos)
}

// LoadAI reads ai.md and parses GFM task list lines.
func (s *Store) LoadAI() ([]model.Todo, error) {
	return s.loadTodoFile(aiFile)
}

// ClearAI removes the ai.md file. Returns nil if the file does not exist.
func (s *Store) ClearAI() error {
	path := filepath.Join(s.basePath, aiFile)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// AIExists reports whether ai.md exists on disk.
func (s *Store) AIExists() bool {
	path := filepath.Join(s.basePath, aiFile)
	_, err := os.Stat(path)
	return err == nil
}

// SaveAI writes AI todos as a GFM task list to ai.md atomically.
func (s *Store) SaveAI(todos []model.Todo) error {
	return s.saveTodoFile(aiFile, todos)
}

// LoadTitle reads the title from title.txt.
// Returns an empty string if the file does not exist.
func (s *Store) LoadTitle() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.basePath, titleFile))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SaveTitle persists a title to title.txt. An empty string removes the file.
func (s *Store) SaveTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		path := filepath.Join(s.basePath, titleFile)
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := s.ensureDir(); err != nil {
		return err
	}
	return s.atomicWrite(titleFile, title)
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
		if base == todosFile || base == aiFile {
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
		if _, err := os.Stat(filepath.Join(s.basePath, name+".md")); err == nil {
			return "", fmt.Errorf("could not generate unique note name (too many collisions)")
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
	return atomicWriteFile(filepath.Join(s.basePath, filename), []byte(data), 0o600)
}

// atomicWriteFile writes data to path via a temporary file and atomic rename.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if tmp != "" {
			os.Remove(tmp)
		}
	}()
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	tmp = "" // disarm defer; rename succeeded
	return nil
}
