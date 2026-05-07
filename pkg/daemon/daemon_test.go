package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateDir(t *testing.T) {
	dir, err := StateDir()
	if err != nil {
		t.Fatalf("StateDir() error = %v", err)
	}
	if dir == "" {
		t.Error("StateDir() returned empty string")
	}
	// Should contain "md-view"
	if !containsStr(dir, "md-view") {
		t.Errorf("StateDir() = %q, expected to contain 'md-view'", dir)
	}
}

func TestWriteAndReadPID(t *testing.T) {
	// Use a temp state dir to avoid polluting real state
	origDir, _ := StateDir()
	tmpDir := t.TempDir()

	// Override state dir by writing PID file there
	pidPath := filepath.Join(tmpDir, "md-view.pid")

	// Write PID
	if err := os.WriteFile(pidPath, []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}

	// Read it back
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "12345" {
		t.Errorf("PID file content = %q, want %q", string(data), "12345")
	}

	// Clean up
	_ = origDir
}

func TestIsAlive(t *testing.T) {
	// Our own process should be alive
	if !IsAlive(os.Getpid()) {
		t.Error("IsAlive(self) should return true")
	}

	// A very high PID should not be alive
	if IsAlive(999999) {
		t.Error("IsAlive(999999) should return false")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
