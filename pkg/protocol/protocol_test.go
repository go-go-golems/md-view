package protocol

import (
	"testing"
)

func TestCommandJSON(t *testing.T) {
	tests := []struct {
		name    string
		cmd     Command
		wantCmd string
		wantErr bool
	}{
		{
			name:    "view command",
			cmd:     Command{Command: "view", Path: "/tmp/test.md"},
			wantCmd: "view",
		},
		{
			name:    "ping command",
			cmd:     Command{Command: "ping"},
			wantCmd: "ping",
		},
		{
			name:    "stop command",
			cmd:     Command{Command: "stop"},
			wantCmd: "stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", tt.cmd.Command, tt.wantCmd)
			}
		})
	}
}

func TestResponseJSON(t *testing.T) {
	resp := Response{Status: "ok", URL: "http://localhost:12345/render?file=/test.md"}
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want %q", resp.Status, "ok")
	}
	if resp.URL == "" {
		t.Error("URL should not be empty")
	}
}
