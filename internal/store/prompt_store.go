package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bvalentino/wtpad/internal/model"
)

// PromptStore handles reading and writing prompt files
// in a shared directory (e.g. ~/.wtpad/prompts/).
type PromptStore struct {
	basePath string
}

// NewPromptStore creates a PromptStore rooted at dir.
// It does not create the directory; that happens lazily on first save.
func NewPromptStore(dir string) *PromptStore {
	return &PromptStore{basePath: dir}
}

// validPromptName checks that name does not escape basePath.
func (ps *PromptStore) validPromptName(name string) error {
	if name == "" {
		return fmt.Errorf("empty prompt name")
	}
	p := filepath.Join(ps.basePath, name+".md")
	cleaned := filepath.Clean(p)
	if !strings.HasPrefix(cleaned, filepath.Clean(ps.basePath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid prompt name: %q", name)
	}
	return nil
}

// ListPrompts scans the prompts directory for .md files.
// Returns prompts sorted by filename descending (newest first).
// Only reads metadata — use LoadPrompt to get the body.
func (ps *PromptStore) ListPrompts() ([]model.Note, error) {
	entries, err := filepath.Glob(filepath.Join(ps.basePath, "*.md"))
	if err != nil {
		return nil, err
	}

	var prompts []model.Note
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		createdAt, _ := time.Parse(noteTimeFmt, name)
		prompts = append(prompts, model.Note{
			Name:      name,
			CreatedAt: createdAt,
		})
	}

	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].Name > prompts[j].Name
	})

	return prompts, nil
}

// LoadPrompt reads a single prompt file by name (without .md extension).
func (ps *PromptStore) LoadPrompt(name string) (*model.Note, error) {
	if err := ps.validPromptName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(ps.basePath, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse(noteTimeFmt, name)

	return &model.Note{
		Name:      name,
		Body:      string(data),
		CreatedAt: createdAt,
	}, nil
}

// SavePrompt writes a prompt atomically. If name is empty, generates a timestamp name.
// Returns the name used.
func (ps *PromptStore) SavePrompt(name string, body string) (string, error) {
	if err := os.MkdirAll(ps.basePath, 0o700); err != nil {
		return "", err
	}

	if name == "" {
		name = time.Now().Format(noteTimeFmt)
		if _, err := os.Stat(filepath.Join(ps.basePath, name+".md")); err == nil {
			for i := 1; i < 100; i++ {
				candidate := fmt.Sprintf("%s-%d", name, i)
				if _, err := os.Stat(filepath.Join(ps.basePath, candidate+".md")); os.IsNotExist(err) {
					name = candidate
					break
				}
			}
		}
		if _, err := os.Stat(filepath.Join(ps.basePath, name+".md")); err == nil {
			return "", fmt.Errorf("could not generate unique prompt name (too many collisions)")
		}
	}

	if err := ps.validPromptName(name); err != nil {
		return "", err
	}

	if err := atomicWriteFile(filepath.Join(ps.basePath, name+".md"), []byte(body), 0o600); err != nil {
		return "", err
	}
	return name, nil
}

// DeletePrompt removes a prompt file.
func (ps *PromptStore) DeletePrompt(name string) error {
	if err := ps.validPromptName(name); err != nil {
		return err
	}
	return os.Remove(filepath.Join(ps.basePath, name+".md"))
}
