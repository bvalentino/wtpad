package tui

import (
	"log"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/bvalentino/wtpad/internal/store"
)

// aiFileChangedMsg signals that ai.md was created, modified, or removed on disk.
type aiFileChangedMsg struct{}

// titleFileChangedMsg signals that title.txt was created, modified, or removed on disk.
type titleFileChangedMsg struct{}

// dirWatcherChannels holds the event channels returned by startDirWatcher.
type dirWatcherChannels struct {
	ai    <-chan aiFileChangedMsg
	title <-chan titleFileChangedMsg
}

// startDirWatcher creates a long-lived fsnotify watcher on the store's directory
// and returns channels that emit events for ai.md and title.txt changes.
// The watcher goroutine runs until the channels are closed (on process exit).
// Returns zero-value channels if the watcher cannot be started.
func startDirWatcher(s *store.Store) dirWatcherChannels {
	// Ensure the directory exists so the watcher has something to watch.
	// Uses EnsureDir (not raw os.MkdirAll) so .wtpad/ is added to .git/info/exclude.
	if err := s.EnsureDir(); err != nil {
		log.Printf("wtpad: cannot create %s for watcher: %v", s.Dir(), err)
		return dirWatcherChannels{}
	}
	dir := s.Dir()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("wtpad: fsnotify watcher failed to start: %v", err)
		return dirWatcherChannels{}
	}

	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		log.Printf("wtpad: fsnotify cannot watch %s: %v", dir, err)
		return dirWatcherChannels{}
	}

	const writeMask = fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename

	aiCh := make(chan aiFileChangedMsg, 1)
	titleCh := make(chan titleFileChangedMsg, 1)
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&writeMask == 0 {
					continue
				}
				switch filepath.Base(event.Name) {
				case "ai.md":
					aiCh <- aiFileChangedMsg{}
				case "title.txt":
					titleCh <- titleFileChangedMsg{}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return dirWatcherChannels{ai: aiCh, title: titleCh}
}

// waitForChange returns a tea.Cmd that blocks until the next event arrives on ch.
// Calling this repeatedly drains the channel one event at a time, keeping the
// single long-lived watcher goroutine alive across events.
func waitForChange[T tea.Msg](ch <-chan T) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
