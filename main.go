package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bvalentino/wtpad/internal/model"
	"github.com/bvalentino/wtpad/internal/store"
)

func main() {
	args := os.Args[1:]

	// No args → TUI (ticket 05-tui-root.md)
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "TUI not yet implemented")
		os.Exit(1)
	}

	s, err := store.New(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
	case "--help", "-h", "help":
		printUsage()
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

	todos = append(todos, model.Todo{Text: text, Done: false})
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
		if !t.Done {
			n++
			fmt.Printf("%d. %s\n", n, t.Text)
		}
	}

	for _, t := range todos {
		if t.Done {
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
		if !t.Done {
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

	todos[found].Done = true
	if err := s.SaveTodos(todos); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done: %s\n", todos[found].Text)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: wtpad [command]

Commands:
  add <text>    Add a todo
  ls            List todos
  note <text>   Create a new note
  done <n>      Mark todo #n done

Run without arguments to start the TUI.`)
}
