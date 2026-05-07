package renderer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantBody  string
		wantHasFM bool
	}{
		{
			name:      "no frontmatter",
			input:     "# Hello\n\nWorld",
			wantBody:  "# Hello\n\nWorld",
			wantHasFM: false,
		},
		{
			name:      "simple frontmatter",
			input:     "---\nTitle: Test Doc\n---\n# Hello\n\nWorld",
			wantTitle: "Test Doc",
			wantBody:  "# Hello\n\nWorld",
			wantHasFM: true,
		},
		{
			name:      "frontmatter with nested values",
			input:     "---\nTitle: My Page\nStatus: active\nTopics:\n  - go\n  - cli\n---\n# Content",
			wantTitle: "My Page",
			wantBody:  "# Content",
			wantHasFM: true,
		},
		{
			name:      "unclosed frontmatter",
			input:     "---\nTitle: Broken\n# Not frontmatter",
			wantBody:  "---\nTitle: Broken\n# Not frontmatter",
			wantHasFM: false,
		},
		{
			name:      "quoted title",
			input:     "---\nTitle: \"Quoted Title\"\n---\nBody",
			wantTitle: "Quoted Title",
			wantBody:  "Body",
			wantHasFM: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, hasFM := extractFrontmatter([]byte(tt.input))
			if hasFM != tt.wantHasFM {
				t.Fatalf("hasFrontmatter = %v, want %v", hasFM, tt.wantHasFM)
			}
			if !hasFM {
				if string(body) != tt.wantBody {
					t.Errorf("body = %q, want %q", string(body), tt.wantBody)
				}
				return
			}
			if fm.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", fm.Title, tt.wantTitle)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}
		})
	}
}

func TestRender(t *testing.T) {
	// Create a temp markdown file
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")
	content := "# Hello\n\nThis is **bold** text.\n"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := Render(mdFile, Options{NoReload: true})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Check key HTML elements
	checks := []string{
		"md-view: test.md",
		"<h1>Hello</h1>",
		"<strong>bold</strong>",
		"markdown-body",
	}
	for _, check := range checks {
		if !contains(html, check) {
			t.Errorf("Render() missing %q in output", check)
		}
	}

	// Should NOT have reload script when NoReload is true
	if contains(html, "MDSReloader") {
		t.Error("Render() should not include reload script when NoReload=true")
	}
}

func TestRenderWithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "fm.md")
	content := "---\nTitle: My Document\nStatus: draft\n---\n# Content\n"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := Render(mdFile, Options{NoReload: true})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Title should come from frontmatter
	if !contains(html, "md-view: My Document") {
		t.Error("Render() should use frontmatter Title as page title")
	}

	// Frontmatter should be in a <details> block
	if !contains(html, "md-view-frontmatter") {
		t.Error("Render() should include frontmatter details block")
	}

	// Body should not contain the frontmatter YAML
	if contains(html, "---") {
		t.Error("Render() should strip frontmatter delimiters from body")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
