package main

import (
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: wtpad add <text>")
		os.Exit(1)
	}

	todos, err := s.LoadTodos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	todos = append(todos, model.Todo{Text: text})
	if err := s.SaveTodos(todos); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added: %s\n", text)
}

func cmdLs(s *store.Store) {
	todos, err := s.LoadTodos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: wtpad note <text>")
		os.Exit(1)
	}

	name, err := s.SaveNote("", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saved note: %s.md\n", name)
}

func cmdDone(s *store.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: wtpad done <n>")
		os.Exit(1)
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		fmt.Fprintf(os.Stderr, "Error: %q is not a valid todo number\n", args[0])
		os.Exit(1)
	}

	todos, err := s.LoadTodos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error: no open todo #%d (have %d open)\n", n, openCount)
		os.Exit(1)
	}

	todos[found].Status = model.StatusDone
	if err := s.SaveTodos(todos); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done: %s\n", todos[found].Text)
}

func cmdTitle(s *store.Store, args []string) {
	if len(args) == 0 {
		title, err := s.LoadTitle()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Title cleared.")
		return
	}

	title := strings.Join(args, " ")
	if runes := []rune(title); len(runes) > 40 {
		fmt.Fprintf(os.Stderr, "Error: title too long (max 40 characters)\n")
		os.Exit(1)
	}
	if err := s.SaveTitle(title); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Title set: %s\n", title)
}

func runTUI(s *store.Store) {
	todos, err := s.LoadTodos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	notes, err := s.ListNotes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	ts := store.NewTemplateStore(filepath.Join(home, ".wtpad", "templates"))
	ps := store.NewPromptStore(filepath.Join(home, ".wtpad", "prompts"))

	prompts, err := ps.ListPrompts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	aiTodos, err := s.LoadAI()
	if err != nil {
		aiTodos = nil // non-fatal: AI tab just won't have data
	}

	branch := gitutil.DetectBranch(".")
	title, err := s.LoadTitle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Usage: wtpad ai %s <text>\n", cmd)
		os.Exit(1)
	}
	todos, err := s.LoadAI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	todos = append(todos, model.Todo{Text: text, Status: status})
	if err := s.SaveAI(todos); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %s\n", verb, text)
}

func cmdAIDone(s *store.Store, args []string) {
	text := strings.Join(args, " ")
	if text == "" {
		fmt.Fprintln(os.Stderr, "Usage: wtpad ai done <text>")
		os.Exit(1)
	}
	todos, err := s.LoadAI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "No matching open or in-progress task: %s\n", text)
		os.Exit(1)
	}
	if err := s.SaveAI(todos); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done: %s\n", text)
}

func cmdAILs(s *store.Store) {
	todos, err := s.LoadAI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(todos) == 0 {
		fmt.Println("No AI tasks.")
		return
	}
	for _, t := range todos {
		switch t.Status {
		case model.StatusDone:
			fmt.Printf("- [x] %s\n", t.Text)
		case model.StatusInProgress:
			fmt.Printf("- [~] %s\n", t.Text)
		default:
			fmt.Printf("- [ ] %s\n", t.Text)
		}
	}
}

func cmdAIClear(s *store.Store) {
	if err := s.ClearAI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("AI tasks cleared.")
}

func cmdAIInstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: wtpad ai install claude-code")
		os.Exit(1)
	}
	switch args[0] {
	case "claude-code":
		installClaudeCode()
	default:
		fmt.Fprintf(os.Stderr, "Unknown integration: %s\n\n", args[0])
		fmt.Fprintln(os.Stderr, "Supported integrations: claude-code")
		os.Exit(1)
	}
}

const localMDMarker = "<!-- wtpad:ai -->"

const localMDContent = localMDMarker + `
## wtpad AI Task Tracking

Track your work using ` + "`wtpad ai`" + ` commands so progress is visible in the wtpad TUI:

- Before starting work: ` + "`wtpad ai start \"task description\"`" + `
- After completing work: ` + "`wtpad ai done \"task description\"`" + `
- To add a task for later: ` + "`wtpad ai add \"task description\"`" + `

Always start a task before doing the work so the user can see what you're working on in real time.
Keep descriptions short — they display in a narrow terminal pane.
`

func installClaudeCode() {
	if err := os.MkdirAll(".claude", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .claude/: %v\n", err)
		os.Exit(1)
	}

	if err := writeLocalMD(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing .claude/local.md: %v\n", err)
		os.Exit(1)
	}

	if err := mergeSettingsHook(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing .claude/settings.local.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Claude Code integration installed:")
	fmt.Println("  .claude/local.md             — AI task tracking instructions")
	fmt.Println("  .claude/settings.local.json  — SessionStart hook")
}

func writeLocalMD() error {
	path := filepath.Join(".claude", "local.md")
	existing, err := os.ReadFile(path)
	if err == nil && strings.Contains(string(existing), localMDMarker) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}
	if len(existing) > 0 {
		prefix += "\n"
	}
	_, err = f.WriteString(prefix + localMDContent)
	return err
}

func mergeSettingsHook() error {
	path := filepath.Join(".claude", "settings.local.json")
	hookCmd := "wtpad ai ls 2>/dev/null || true"

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

	for _, h := range sessionStart {
		if hm, ok := h.(map[string]any); ok {
			if cmd, ok := hm["command"].(string); ok && cmd == hookCmd {
				return nil
			}
		}
	}

	sessionStart = append(sessionStart, map[string]any{
		"type":    "command",
		"command": hookCmd,
	})
	hooks["SessionStart"] = sessionStart
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func printAIUsage() {
	fmt.Fprintln(os.Stderr, `Usage: wtpad ai <command>

Commands:
  add <text>              Add an open task
  start <text>            Add an in-progress task
  done <text>             Mark a task as done
  ls                      List AI tasks
  clear                   Remove all AI tasks
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
