package renderer

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// frontmatterData holds parsed YAML frontmatter key-value pairs.
type frontmatterData struct {
	Title   string
	Entries []fmEntry
}

type fmEntry struct {
	Key   string
	Value string
}

// extractFrontmatter splits input into YAML frontmatter and body.
// Returns (frontmatterData, body, hasFrontmatter).
func extractFrontmatter(data []byte) (*frontmatterData, []byte, bool) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return nil, data, false
	}

	end := strings.Index(content[3:], "\n---")
	if end == -1 {
		return nil, data, false
	}

	rawFM := content[3 : 3+end]
	body := content[3+end+4:]
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	fm := parseFrontmatter(rawFM)
	return fm, []byte(body), true
}

// parseFrontmatter parses simple YAML key: value pairs from frontmatter.
// Handles nested structures (lists, maps) by detecting indentation.
func parseFrontmatter(raw string) *frontmatterData {
	data := &frontmatterData{}
	lines := strings.Split(raw, "\n")

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Top-level key: value
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			colonIdx := strings.Index(line, ":")
			if colonIdx == -1 {
				i++
				continue
			}

			key := strings.TrimSpace(line[:colonIdx])
			value := strings.TrimSpace(line[colonIdx+1:])

			// If value is empty, collect nested content
			if value == "" {
				// Collect all indented lines below as the value
				var nested []string
				i++
				for i < len(lines) && (strings.HasPrefix(lines[i], "  ") || strings.HasPrefix(lines[i], "\t") || strings.TrimSpace(lines[i]) == "") {
					nested = append(nested, lines[i])
					i++
				}
				value = strings.Join(nested, "\n")
			} else {
				// Strip surrounding quotes
				value = stripQuotes(value)
				i++
			}

			// Track title
			if key == "Title" && data.Title == "" {
				data.Title = stripQuotes(value)
			}

			data.Entries = append(data.Entries, fmEntry{Key: key, Value: value})
		} else {
			i++
		}
	}

	return data
}

var reQuoted = regexp.MustCompile(`^"(.*)"$`)

func stripQuotes(s string) string {
	if m := reQuoted.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	// Also strip single quotes
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

// formatFrontmatterHTML renders YAML frontmatter as a collapsible <details> block
// with a formatted key-value display.
func formatFrontmatterHTML(fm *frontmatterData) string {
	var buf strings.Builder

	buf.WriteString(`<details class="md-view-frontmatter">
<summary>Frontmatter</summary>
<div class="md-view-fm-table">
`)

	for _, entry := range fm.Entries {
		key := htmlEscape(entry.Key)
		value := htmlEscape(strings.TrimSpace(entry.Value))

		// Check if value is multi-line (nested YAML)
		if strings.Contains(value, "\n") {
			buf.WriteString(fmt.Sprintf(`<div class="md-view-fm-row">
<span class="md-view-fm-key">%s</span>
<pre class="md-view-fm-value">%s</pre>
</div>
`, key, value))
		} else {
			buf.WriteString(fmt.Sprintf(`<div class="md-view-fm-row">
<span class="md-view-fm-key">%s</span>
<span class="md-view-fm-value">%s</span>
</div>
`, key, value))
		}
	}

	buf.WriteString(`</div>
</details>`)
	return buf.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

const frontmatterCSS = `
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
.md-view-fm-table {
    border-top: 1px solid #d0d7de;
    font-size: 13px;
    line-height: 1.5;
}
.md-view-fm-row {
    display: flex;
    border-bottom: 1px solid #eaeef2;
    align-items: baseline;
}
.md-view-fm-row:last-child {
    border-bottom: none;
}
.md-view-fm-key {
    min-width: 120px;
    padding: 6px 12px;
    font-weight: 600;
    color: #24292e;
    background: #f0f2f4;
    flex-shrink: 0;
}
.md-view-fm-value {
    padding: 6px 12px;
    color: #656d76;
    word-break: break-word;
}
.md-view-fm-value pre {
    margin: 0;
    padding: 0;
    font-size: 12px;
    line-height: 1.4;
    background: transparent;
    white-space: pre-wrap;
}
`

// Render reads a markdown file and returns full HTML.
func Render(filePath string, opts Options) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	// Extract frontmatter
	fm, body, hasFM := extractFrontmatter(data)

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

	// Determine page title: explicit > frontmatter Title > filename
	title := opts.Title
	if title == "" && hasFM && fm.Title != "" {
		title = fm.Title
	}
	if title == "" {
		title = filepath.Base(filePath)
	}
	title = "md-view: " + title

	// Build frontmatter section
	fmHTML := ""
	if hasFM {
		fmHTML = formatFrontmatterHTML(fm)
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
%s
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
		frontmatterCSS,
		fmHTML,
		buf.String(),
		reloadScript,
	)

	return htmlPage, nil
}
