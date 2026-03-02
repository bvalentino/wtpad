package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bvalentino/wtpad/internal/model"
)

// TemplateInfo holds metadata about a template file.
type TemplateInfo struct {
	Name      string
	TodoCount int
}

// TemplateStore handles reading and writing todo templates
// in a shared directory (e.g. ~/.wtpad/templates/).
type TemplateStore struct {
	basePath string
}

// NewTemplateStore creates a TemplateStore rooted at dir.
// It does not create the directory; that happens lazily on first save.
func NewTemplateStore(dir string) *TemplateStore {
	return &TemplateStore{basePath: dir}
}

// validTemplateName checks that name does not escape basePath.
func (ts *TemplateStore) validTemplateName(name string) error {
	if name == "" {
		return fmt.Errorf("empty template name")
	}
	p := filepath.Join(ts.basePath, name+".md")
	cleaned := filepath.Clean(p)
	if !strings.HasPrefix(cleaned, filepath.Clean(ts.basePath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid template name: %q", name)
	}
	return nil
}

// ListTemplates scans the templates directory for .md files.
// Returns templates sorted alphabetically by name.
func (ts *TemplateStore) ListTemplates() ([]TemplateInfo, error) {
	entries, err := filepath.Glob(filepath.Join(ts.basePath, "*.md"))
	if err != nil {
		return nil, err
	}

	var templates []TemplateInfo
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		count := countTodoLines(path)
		templates = append(templates, TemplateInfo{
			Name:      name,
			TodoCount: count,
		})
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// countTodoLines counts GFM task list lines in a file.
func countTodoLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if _, ok := parseTodoLine(scanner.Text()); ok {
			count++
		}
	}
	return count
}

// LoadTemplate reads a template file and returns its todos.
func (ts *TemplateStore) LoadTemplate(name string) ([]model.Todo, error) {
	if err := ts.validTemplateName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(ts.basePath, name+".md")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var todos []model.Todo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if todo, ok := parseTodoLine(scanner.Text()); ok {
			todos = append(todos, todo)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

// SaveTemplate writes todos as a GFM task list to a template file.
// Returns the name used.
func (ts *TemplateStore) SaveTemplate(name string, todos []model.Todo) (string, error) {
	if err := ts.validTemplateName(name); err != nil {
		return "", err
	}

	if err := os.MkdirAll(ts.basePath, 0o755); err != nil {
		return "", err
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

	target := filepath.Join(ts.basePath, name+".md")
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, []byte(buf.String()), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return name, nil
}

// DeleteTemplate removes a template file.
func (ts *TemplateStore) DeleteTemplate(name string) error {
	if err := ts.validTemplateName(name); err != nil {
		return err
	}
	return os.Remove(filepath.Join(ts.basePath, name+".md"))
}

// TemplateExists checks whether a template with the given name exists.
func (ts *TemplateStore) TemplateExists(name string) bool {
	if err := ts.validTemplateName(name); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(ts.basePath, name+".md"))
	return err == nil
}
