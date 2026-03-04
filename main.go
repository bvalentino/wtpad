package main

import (
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

Run without arguments to start the TUI.`)
}
