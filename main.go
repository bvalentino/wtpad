package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	gitutil "github.com/bvalentino/wtpad/internal/git"
	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
	"github.com/bvalentino/wtpad/internal/tui"
)

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
			printUsage()
			return
		}
	}

	s, err := store.New(".")
	if err != nil {
		fatal("Error: %v", err)
	}

	// No args → TUI
	if len(args) == 0 {
		runTUI(s)
		return
	}

	switch args[0] {
	case "add":
		cmdAdd(s, args[1:])
	case "ls":
		cmdLs(s)
	case "note":
		cmdNote(s, args[1:])
	case "done":
		cmdDone(s, args[1:])
	case "ai":
		cmdAI(s, args[1:])
	case "title":
		cmdTitle(s, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func cmdAdd(s *store.Store, args []string) {
	text := strings.Join(args, " ")
	if text == "" {
		fatal("Usage: wtpad add <text>")
	}

	todos, err := s.LoadTodos()
	if err != nil {
		fatal("Error: %v", err)
	}

	todos = append(todos, model.Todo{Text: text})
	if err := s.SaveTodos(todos); err != nil {
		fatal("Error: %v", err)
	}

	fmt.Printf("Added: %s\n", text)
}

func cmdLs(s *store.Store) {
	todos, err := s.LoadTodos()
	if err != nil {
		fatal("Error: %v", err)
	}

	if len(todos) == 0 {
		fmt.Println("No todos.")
		return
	}

	n := 0
	for _, t := range todos {
		if t.Status != model.StatusDone {
			n++
			prefix := fmt.Sprintf("%d.", n)
			if t.Status == model.StatusInProgress {
				prefix = fmt.Sprintf("%d.~", n)
			}
			fmt.Printf("%s %s\n", prefix, t.Text)
		}
	}

	for _, t := range todos {
		if t.Status == model.StatusDone {
			fmt.Printf("✓ %s\n", t.Text)
		}
	}
}

func cmdNote(s *store.Store, args []string) {
	body := strings.Join(args, " ")
	if body == "" {
		fatal("Usage: wtpad note <text>")
	}

	name, err := s.SaveNote("", body)
	if err != nil {
		fatal("Error: %v", err)
	}

	fmt.Printf("Saved note: %s.md\n", name)
}

func cmdDone(s *store.Store, args []string) {
	if len(args) == 0 {
		fatal("Usage: wtpad done <n>")
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		fatal("Error: %q is not a valid todo number", args[0])
	}

	todos, err := s.LoadTodos()
	if err != nil {
		fatal("Error: %v", err)
	}

	// Find the Nth open todo (same order as ls)
	openCount := 0
	found := -1
	for i, t := range todos {
		if t.Status != model.StatusDone {
			openCount++
			if openCount == n {
				found = i
				break
			}
		}
	}

	if found == -1 {
		fatal("Error: no open todo #%d (have %d open)", n, openCount)
	}

	todos[found].Status = model.StatusDone
	if err := s.SaveTodos(todos); err != nil {
		fatal("Error: %v", err)
	}

	fmt.Printf("Done: %s\n", todos[found].Text)
}

func cmdTitle(s *store.Store, args []string) {
	if len(args) == 0 {
		title, err := s.LoadTitle()
		if err != nil {
			fatal("Error: %v", err)
		}
		if title == "" {
			fmt.Println("No title set.")
		} else {
			fmt.Println(title)
		}
		return
	}

	if args[0] == "--clear" {
		if err := s.SaveTitle(""); err != nil {
			fatal("Error: %v", err)
		}
		fmt.Println("Title cleared.")
		return
	}

	title := strings.Join(args, " ")
	if runes := []rune(title); len(runes) > 40 {
		fatal("Error: title too long (max 40 characters)")
	}
	if err := s.SaveTitle(title); err != nil {
		fatal("Error: %v", err)
	}
	fmt.Printf("Title set: %s\n", title)
}

func runTUI(s *store.Store) {
	todos, err := s.LoadTodos()
	if err != nil {
		fatal("Error: %v", err)
	}

	notes, err := s.ListNotes()
	if err != nil {
		fatal("Error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fatal("Error: %v", err)
	}
	ts := store.NewTemplateStore(filepath.Join(home, ".wtpad", "templates"))
	ps := store.NewPromptStore(filepath.Join(home, ".wtpad", "prompts"))

	prompts, err := ps.ListPrompts()
	if err != nil {
		fatal("Error: %v", err)
	}

	aiTodos, err := s.LoadAI()
	if err != nil {
		aiTodos = nil // non-fatal: AI tab just won't have data
	}

	branch := gitutil.DetectBranch(".")
	title, err := s.LoadTitle()
	if err != nil {
		fatal("Error: %v", err)
	}

	app := tui.New(tui.AppConfig{
		Store:         s,
		TemplateStore: ts,
		PromptStore:   ps,
		Todos:         todos,
		Notes:         notes,
		Prompts:       prompts,
		AITodos:       aiTodos,
		Branch:        branch,
		Title:         title,
	})
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fatal("Error: %v", err)
	}
}

func cmdAI(s *store.Store, args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printAIUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}
	switch args[0] {
	case "add":
		cmdAIAdd(s, args[1:])
	case "start":
		cmdAIStart(s, args[1:])
	case "done":
		cmdAIDone(s, args[1:])
	case "ls":
		cmdAILs(s)
	case "clear":
		cmdAIClear(s)
	case "prompt":
		cmdAIPrompt(s)
	case "install":
		cmdAIInstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown ai command: %s\n\n", args[0])
		printAIUsage()
		os.Exit(1)
	}
}

func cmdAIAdd(s *store.Store, args []string) {
	aiAppendTask(s, args, model.StatusOpen, "add", "Added")
}

func cmdAIStart(s *store.Store, args []string) {
	aiAppendTask(s, args, model.StatusInProgress, "start", "Started")
}

func aiAppendTask(s *store.Store, args []string, status model.TodoStatus, cmd, verb string) {
	text := strings.Join(args, " ")
	if text == "" {
		fatal("Usage: wtpad ai %s <text>", cmd)
	}
	todos, err := s.LoadAI()
	if err != nil {
		fatal("Error: %v", err)
	}
	todos = append(todos, model.Todo{Text: text, Status: status})
	if err := s.SaveAI(todos); err != nil {
		fatal("Error: %v", err)
	}
	fmt.Printf("%s: %s\n", verb, text)
}

func cmdAIDone(s *store.Store, args []string) {
	text := strings.Join(args, " ")
	if text == "" {
		fatal("Usage: wtpad ai done <text>")
	}
	todos, err := s.LoadAI()
	if err != nil {
		fatal("Error: %v", err)
	}
	found := false
	for i, t := range todos {
		if strings.EqualFold(t.Text, text) && t.Status != model.StatusDone {
			todos[i].Status = model.StatusDone
			found = true
			break
		}
	}
	if !found {
		fatal("No matching open or in-progress task: %s", text)
	}
	if err := s.SaveAI(todos); err != nil {
		fatal("Error: %v", err)
	}
	fmt.Printf("Done: %s\n", text)
}

func cmdAILs(s *store.Store) {
	todos, err := s.LoadAI()
	if err != nil {
		fatal("Error: %v", err)
	}
	if len(todos) == 0 {
		fmt.Println("No AI tasks.")
		return
	}
	for _, t := range todos {
		fmt.Println(t.GFMLine())
	}
}

func cmdAIClear(s *store.Store) {
	if err := s.ClearAI(); err != nil {
		fatal("Error: %v", err)
	}
	fmt.Println("AI tasks cleared.")
}

const promptText = `## wtpad AI Task Tracking

Track your work using ` + "`wtpad ai`" + ` commands so progress is visible in the wtpad TUI:

- Before starting work: ` + "`wtpad ai start \"task description\"`" + `
- After completing work: ` + "`wtpad ai done \"task description\"`" + `
- To add a task for later: ` + "`wtpad ai add \"task description\"`" + `

Always start a task before doing the work so the user can see what you're working on in real time.
Keep descriptions short — they display in a narrow terminal pane.`

func cmdAIPrompt(s *store.Store) {
	fmt.Println(promptText)
	todos, err := s.LoadAI()
	if err != nil || len(todos) == 0 {
		return
	}
	fmt.Println("\nCurrent AI tasks:")
	for _, t := range todos {
		fmt.Println(t.GFMLine())
	}
}

func cmdAIInstall(args []string) {
	if len(args) == 0 {
		fatal("Usage: wtpad ai install claude-code")
	}
	switch args[0] {
	case "claude-code":
		installClaudeCode()
	default:
		fatal("Unknown integration: %s\n\nSupported integrations: claude-code", args[0])
	}
}

func installClaudeCode() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("Error finding home directory: %v", err)
	}
	claudeDir := filepath.Join(home, ".claude")

	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		fatal("Error creating ~/.claude/: %v", err)
	}

	if err := mergeSettingsHook(claudeDir); err != nil {
		fatal("Error writing ~/.claude/settings.json: %v", err)
	}

	fmt.Println("Claude Code integration installed:")
	fmt.Println("  ~/.claude/settings.json  — SessionStart hook (wtpad ai prompt)")
}

func mergeSettingsHook(claudeDir string) error {
	path := filepath.Join(claudeDir, "settings.json")
	hookCmd := `command -v wtpad >/dev/null 2>&1 && wtpad ai prompt 2>/dev/null || { echo "wtpad: command not found — remove this hook from ~/.claude/settings.json to clean up"; true; }`

	var settings map[string]any
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	var hooks map[string]any
	switch v := settings["hooks"].(type) {
	case map[string]any:
		hooks = v
	case nil:
		hooks = make(map[string]any)
	default:
		return fmt.Errorf("%s: \"hooks\" has unexpected type %T", path, v)
	}

	var sessionStart []any
	switch v := hooks["SessionStart"].(type) {
	case []any:
		sessionStart = v
	case nil:
		// no existing hooks
	default:
		return fmt.Errorf("%s: \"hooks.SessionStart\" has unexpected type %T", path, v)
	}

	// Look for an existing wtpad hook in both old (flat) and new (nested) formats.
	found := false
	for i, entry := range sessionStart {
		em, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		// Old flat format: {"type":"command","command":"wtpad ai ls ..."}
		if cmd, ok := em["command"].(string); ok && strings.Contains(cmd, "wtpad ai") {
			// Replace the flat entry with the new nested format
			sessionStart[i] = map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": hookCmd,
					},
				},
			}
			found = true
			break
		}

		// New nested format: {"hooks":[{"type":"command","command":"wtpad ai ls ..."}]}
		hooksArr, _ := em["hooks"].([]any)
		for _, h := range hooksArr {
			if hm, ok := h.(map[string]any); ok {
				if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "wtpad ai") {
					if cmd == hookCmd {
						return nil // already up to date
					}
					hm["command"] = hookCmd
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		sessionStart = append(sessionStart, map[string]any{
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": hookCmd,
				},
			},
		})
		hooks["SessionStart"] = sessionStart
		settings["hooks"] = hooks
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		return err
	}
	return store.AtomicWriteFile(path, buf.Bytes(), 0o644)
}

func printAIUsage() {
	fmt.Fprintln(os.Stderr, `Usage: wtpad ai <command>

Commands:
  add <text>              Add an open task
  start <text>            Add an in-progress task
  done <text>             Mark a task as done
  ls                      List AI tasks
  clear                   Remove all AI tasks
  prompt                  Print AI instructions and current tasks
  install claude-code     Set up Claude Code integration`)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: wtpad [command]

Commands:
  add <text>    Add a todo
  ls            List todos
  note <text>   Create a new note
  done <n>      Mark todo #n done
  title <text>  Set a title shown above the logo
  title --clear Remove the title
  title         Show the current title
  ai <command>  AI task tracking (see wtpad ai --help)

Run without arguments to start the TUI.`)
}
