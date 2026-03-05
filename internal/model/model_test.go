package model

import "testing"

func TestGFMLine(t *testing.T) {
	tests := []struct {
		todo Todo
		want string
	}{
		{Todo{Text: "buy milk", Status: StatusOpen}, "- [ ] buy milk"},
		{Todo{Text: "in progress", Status: StatusInProgress}, "- [~] in progress"},
		{Todo{Text: "all done", Status: StatusDone}, "- [x] all done"},
		{Todo{Text: "", Status: StatusOpen}, "- [ ] "},
	}
	for _, tt := range tests {
		got := tt.todo.GFMLine()
		if got != tt.want {
			t.Errorf("GFMLine(%+v) = %q, want %q", tt.todo, got, tt.want)
		}
	}
}
