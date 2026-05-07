package renderer

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	chroma_html "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed static/base.css
var defaultCSS []byte

//go:embed static/reload.js
var reloadJS []byte

// CSS returns the embedded GitHub-flavored CSS.
func CSS() []byte {
	return defaultCSS
}

// ReloadJS returns the embedded live-reload script.
func ReloadJS() []byte {
	return reloadJS
}

// ChromaCSS returns the CSS for syntax highlighting (chroma/github style).
func ChromaCSS() (string, error) {
	formatter := chroma_html.New(chroma_html.WithClasses(true))
	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}
	buf := &bytes.Buffer{}
	if err := formatter.WriteCSS(buf, style); err != nil {
		return "", fmt.Errorf("cannot generate chroma CSS: %w", err)
	}
	return buf.String(), nil
}

// Options for rendering.
type Options struct {
	// NoReload disables SSE live reload injection.
	NoReload bool
	// File is the absolute path of the markdown file (used for SSE endpoint).
	File string
	// Title is the page title (defaults to filename if empty).
	Title string
	// Port is the HTTP port (used for SSE endpoint URL).
	Port int
}

// extractFrontmatter splits input into YAML frontmatter and body.
// Returns (frontmatter, body, hasFrontmatter).
func extractFrontmatter(data []byte) (string, []byte, bool) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return "", data, false
	}

	// Find closing ---
	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		// No closing delimiter — not frontmatter
		return "", data, false
	}

	frontmatter := content[3 : 3+end]
	body := content[3+end+4:] // skip past closing ---

	// Trim leading newlines from body
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	return frontmatter, []byte(body), true
}

// formatFrontmatterHTML renders YAML frontmatter as a collapsible <details> block.
func formatFrontmatterHTML(frontmatter string) string {
	// Escape HTML entities
	fm := htmlEscape(frontmatter)

	// Format YAML nicely: trim trailing whitespace per line, remove blank lines at start/end
	lines := strings.Split(fm, "\n")
	var formatted []string
	for _, line := range lines {
		formatted = append(formatted, strings.TrimRight(line, " \t"))
	}
	// Trim leading/trailing blank lines
	for len(formatted) > 0 && formatted[0] == "" {
		formatted = formatted[1:]
	}
	for len(formatted) > 0 && formatted[len(formatted)-1] == "" {
		formatted = formatted[:len(formatted)-1]
	}

	fmFormatted := strings.Join(formatted, "\n")

	return fmt.Sprintf(`<details class="md-view-frontmatter">
<summary>Frontmatter</summary>
<pre><code class="language-yaml">%s</code></pre>
</details>`, fmFormatted)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// Render reads a markdown file and returns full HTML.
func Render(filePath string, opts Options) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	// Extract frontmatter
	frontmatter, body, hasFM := extractFrontmatter(data)

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					chroma_html.WithClasses(true),
				),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return "", fmt.Errorf("cannot convert markdown: %w", err)
	}

	chromaCSS, err := ChromaCSS()
	if err != nil {
		return "", err
	}

	reloadScript := ""
	if !opts.NoReload && opts.File != "" {
		encodedPath := strings.ReplaceAll(opts.File, " ", "%20")
		reloadScript = fmt.Sprintf(
			`<script>
%s
new MDSReloader("http://localhost:%d/events?file=%s");
</script>`,
			string(reloadJS),
			opts.Port,
			encodedPath,
		)
	}

	title := opts.Title
	if title == "" {
		title = filepath.Base(filePath)
	}
	title = "md-view: " + title

	// Build frontmatter section
	fmHTML := ""
	if hasFM {
		fmHTML = formatFrontmatterHTML(frontmatter)
	}

	htmlPage := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
%s
</style>
<style>
%s
</style>
<style>
.md-view-frontmatter {
    margin-bottom: 24px;
    border: 1px solid #d0d7de;
    border-radius: 6px;
    background: #f6f8fa;
    padding: 0;
}
.md-view-frontmatter > summary {
    padding: 8px 12px;
    cursor: pointer;
    font-size: 13px;
    color: #656d76;
    user-select: none;
    list-style: none;
    display: flex;
    align-items: center;
    gap: 6px;
}
.md-view-frontmatter > summary::before {
    content: "▶";
    font-size: 10px;
    transition: transform 0.15s;
}
.md-view-frontmatter[open] > summary::before {
    transform: rotate(90deg);
}
.md-view-frontmatter > summary:hover {
    background: #eaeef2;
}
.md-view-frontmatter pre {
    margin: 0;
    padding: 12px 16px;
    border-top: 1px solid #d0d7de;
    font-size: 13px;
    line-height: 1.5;
    overflow-x: auto;
}
.md-view-frontmatter code {
    font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
    background: transparent;
    padding: 0;
}
</style>
</head>
<body class="markdown-body">
%s
%s
%s
</body>
</html>`,
		title,
		string(defaultCSS),
		chromaCSS,
		fmHTML,
		buf.String(),
		reloadScript,
	)

	return htmlPage, nil
}
