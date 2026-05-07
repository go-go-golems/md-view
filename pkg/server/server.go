package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-go-golems/md-view/pkg/daemon"
	"github.com/go-go-golems/md-view/pkg/protocol"
	"github.com/go-go-golems/md-view/pkg/renderer"
	"github.com/go-go-golems/md-view/pkg/watcher"
)

// Server is the md-view HTTP + Unix socket server.
type Server struct {
	httpServer *http.Server
	port       int
	watcher    *watcher.FileWatcher
	mu         sync.Mutex
	sseClients map[string]map[<-chan struct{}]struct{} // file path → set of watch channels
	browser    string // override browser command
	noReload   bool   // disable live reload
}

// NewServer creates a new Server bound to localhost on the given port (0 = random).
func NewServer(port int, browser string, noReload bool) (*Server, error) {
	fw, err := watcher.New()
	if err != nil {
		return nil, err
	}

	s := &Server{
		port:       port,
		watcher:    fw,
		sseClients: make(map[string]map[<-chan struct{}]struct{}),
		browser:    browser,
		noReload:   noReload,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/render", s.handleRender)
	mux.HandleFunc("/raw", s.handleRaw)
	mux.HandleFunc("/events", s.handleEvents)
	mux.HandleFunc("/static/", s.handleStatic)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	return s, nil
}

// Start starts the HTTP server and Unix socket listener.
// This is a blocking call — it returns when the server shuts down.
func (s *Server) Start(ctx context.Context) error {
	// Start file watcher
	s.watcher.Start()

	// Start HTTP listener
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", s.httpServer.Addr, err)
	}

	// Get the actual port (important when port=0)
	s.port = listener.Addr().(*net.TCPAddr).Port

	// Write state files
	if err := daemon.WritePort(s.port); err != nil {
		return fmt.Errorf("cannot write port file: %w", err)
	}

	// Start Unix socket listener
	socketPath, err := daemon.SocketPath()
	if err != nil {
		return fmt.Errorf("cannot determine socket path: %w", err)
	}
	// Remove stale socket
	_ = os.Remove(socketPath)

	socketListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("cannot listen on unix socket: %w", err)
	}
	// Restrict socket permissions
	_ = os.Chmod(socketPath, 0600)

	go s.acceptUnixConnections(ctx, socketListener)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case <-sigCh:
			log.Println("Received shutdown signal")
			s.Shutdown()
		case <-ctx.Done():
			s.Shutdown()
		}
	}()

	log.Printf("md-view server listening on http://localhost:%d (socket: %s)", s.port, socketPath)

	// Serve HTTP
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	s.watcher.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.httpServer.Shutdown(ctx)
	_ = daemon.Cleanup()
}

// Port returns the actual HTTP port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// acceptUnixConnections accepts connections on the Unix domain socket.
func (s *Server) acceptUnixConnections(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("Unix socket accept error: %v", err)
				continue
			}
		}
		go s.handleSocketConn(ctx, conn)
	}
}

// handleSocketConn handles a single Unix socket connection.
func (s *Server) handleSocketConn(_ context.Context, conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	var cmd protocol.Command
	line := strings.TrimSpace(string(buf[:n]))
	if line == "" {
		return
	}
	if err := json.Unmarshal([]byte(line), &cmd); err != nil {
		resp := protocol.Response{Status: "error", Message: fmt.Sprintf("invalid command: %v", err)}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
		return
	}

	switch cmd.Command {
	case "view":
		url := fmt.Sprintf("http://localhost:%d/render?file=%s", s.port, urlEncodePath(cmd.Path))
		resp := protocol.Response{Status: "ok", URL: url}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))

		// Open browser in background
		go s.openBrowser(url)

	case "ping":
		resp := protocol.Response{Status: "pong"}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))

	case "stop":
		resp := protocol.Response{Status: "ok", Message: "shutting down"}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.Shutdown()
			os.Exit(0)
		}()

	default:
		resp := protocol.Response{Status: "error", Message: fmt.Sprintf("unknown command: %s", cmd.Command)}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
	}
}

// --- HTTP Handlers ---

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "missing file parameter", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid path: %v", err), http.StatusBadRequest)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot stat file: %v", err), http.StatusNotFound)
		return
	}
	if !info.Mode().IsRegular() {
		http.Error(w, "not a regular file", http.StatusBadRequest)
		return
	}

	opts := renderer.Options{
		File:     absPath,
		Port:     s.port,
		NoReload: s.noReload,
	}

	html, err := renderer.Render(absPath, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("render error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "missing file parameter", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid path: %v", err), http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot read file: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "missing file parameter", http.StatusBadRequest)
		return
	}

	// Resolve absolute path for consistent key
	absPath, _ := filepath.Abs(filePath)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch, err := s.watcher.Watch(absPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot watch file: %v", err), http.StatusInternalServerError)
		return
	}

	// Track SSE client for cleanup
	s.mu.Lock()
	if s.sseClients[absPath] == nil {
		s.sseClients[absPath] = make(map[<-chan struct{}]struct{})
	}
	s.sseClients[absPath][ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sseClients[absPath], ch)
		if len(s.sseClients[absPath]) == 0 {
			delete(s.sseClients, absPath)
		}
		s.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ch:
			fmt.Fprintf(w, "event: reload\ndata: reload\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/static/base.css":
		w.Header().Set("Content-Type", "text/css")
		w.Write(renderer.CSS())
	case "/static/reload.js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(renderer.ReloadJS())
	default:
		http.NotFound(w, r)
	}
}

// --- Helpers ---

func (s *Server) openBrowser(url string) {
	browser := s.browser
	if browser == "" {
		browser = os.Getenv("BROWSER")
	}
	if browser == "" {
		for _, b := range []string{"xdg-open", "firefox", "google-chrome", "chromium"} {
			if _, err := exec.LookPath(b); err == nil {
				browser = b
				break
			}
		}
	}
	if browser == "" {
		log.Println("Warning: no browser found (set $BROWSER)")
		return
	}

	var cmd *exec.Cmd
	if browser == "xdg-open" {
		cmd = exec.Command(browser, url)
	} else {
		cmd = exec.Command(browser, "--new-tab", url)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		log.Printf("Cannot open browser: %v", err)
	}
}

func urlEncodePath(p string) string {
	return strings.ReplaceAll(p, " ", "%20")
}
