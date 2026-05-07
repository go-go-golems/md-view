package renderer

import (
	_ "embed"
	"bytes"
	"fmt"
	"os"
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
	// Port is the HTTP port (used for SSE endpoint URL).
	Port int
}

// Render reads a markdown file and returns full HTML.
func Render(filePath string, opts Options) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

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
	if err := md.Convert(data, &buf); err != nil {
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
</head>
<body class="markdown-body">
%s
%s
</body>
</html>`,
		filePath,
		string(defaultCSS),
		chromaCSS,
		buf.String(),
		reloadScript,
	)

	return htmlPage, nil
}
