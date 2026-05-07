package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Command represents a client → server message.
type Command struct {
	Command string `json:"command"`
	Path    string `json:"path,omitempty"`
	Dark    bool   `json:"dark,omitempty"`
	Browser string `json:"browser,omitempty"`
}

// Response represents a server → client message.
type Response struct {
	Status  string `json:"status"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

// SendCommand sends a command over a Unix socket and returns the response.
func SendCommand(socketPath string, cmd Command) (*Response, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon socket: %w", err)
	}
	defer func() { _ = conn.Close() }()

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal command: %w", err)
	}

	// NDJSON: newline-delimited
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return nil, fmt.Errorf("cannot send command: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("cannot read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, fmt.Errorf("cannot parse response: %w", err)
	}

	return &resp, nil
}
