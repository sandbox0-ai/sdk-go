package sandbox0

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// StreamInput represents a context WebSocket input message.
type StreamInput struct {
	Type      string `json:"type,omitempty"`
	Data      string `json:"data,omitempty"`
	Rows      uint16 `json:"rows,omitempty"`
	Cols      uint16 `json:"cols,omitempty"`
	Signal    string `json:"signal,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// StreamOutput represents streamed process output.
type StreamOutput struct {
	SandboxID string
	ContextID string
	Source    string `json:"source"`
	Data      string `json:"data"`
}

// StreamDone represents stream completion metadata.
type StreamDone struct {
	SandboxID string
	ContextID string
	RequestID string `json:"request_id,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	State     string `json:"state,omitempty"`
}

type wsContextMessage struct {
	Type      string `json:"type,omitempty"`
	Source    string `json:"source,omitempty"`
	Data      string `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	State     string `json:"state,omitempty"`
}

// ContextStream wraps a context WebSocket for structured I/O.
type ContextStream struct {
	SandboxID string
	ContextID string

	conn           *websocket.Conn
	writeMu        sync.Mutex
	requestCounter uint64
	doneMu         sync.RWMutex
	done           *StreamDone
}

func newContextStream(sandboxID, contextID string, conn *websocket.Conn) *ContextStream {
	return &ContextStream{
		SandboxID: sandboxID,
		ContextID: contextID,
		conn:      conn,
	}
}

// Send writes a structured input message to the context stream.
func (s *ContextStream) Send(input StreamInput) error {
	normalized, err := s.normalizeInput(input)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(normalized)
}

// SendInput writes input bytes to the context stream.
func (s *ContextStream) SendInput(data, requestID string) error {
	return s.Send(StreamInput{Type: "input", Data: data, RequestID: requestID})
}

// SendResize resizes the remote PTY.
func (s *ContextStream) SendResize(rows, cols uint16) error {
	return s.Send(StreamInput{Type: "resize", Rows: rows, Cols: cols})
}

// SendSignal forwards a signal request.
func (s *ContextStream) SendSignal(signal string) error {
	return s.Send(StreamInput{Type: "signal", Signal: signal})
}

// Recv reads the next output message from the stream.
func (s *ContextStream) Recv() (StreamOutput, error) {
	for {
		var msg wsContextMessage
		if err := s.conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return StreamOutput{}, io.EOF
			}
			return StreamOutput{}, err
		}
		if isTerminalDoneMessage(msg) {
			s.setDone(StreamDone{
				SandboxID: s.SandboxID,
				ContextID: s.ContextID,
				RequestID: msg.RequestID,
				ExitCode:  msg.ExitCode,
				State:     msg.State,
			})
			return StreamOutput{}, io.EOF
		}
		if msg.Type != "" && msg.Type != "output" {
			continue
		}
		if msg.Source == "" && msg.Data == "" {
			continue
		}
		return StreamOutput{
			SandboxID: s.SandboxID,
			ContextID: s.ContextID,
			Source:    msg.Source,
			Data:      msg.Data,
		}, nil
	}
}

// Result returns the last terminal done message observed by Recv or Wait.
func (s *ContextStream) Result() (StreamDone, bool) {
	s.doneMu.RLock()
	defer s.doneMu.RUnlock()
	if s.done == nil {
		return StreamDone{}, false
	}
	return *s.done, true
}

// Wait reads until the stream reaches a terminal done message or closes.
func (s *ContextStream) Wait() (StreamDone, error) {
	if done, ok := s.Result(); ok {
		return done, nil
	}
	for {
		_, err := s.Recv()
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			if done, ok := s.Result(); ok {
				return done, nil
			}
			return StreamDone{SandboxID: s.SandboxID, ContextID: s.ContextID}, nil
		}
		return StreamDone{}, err
	}
}

// WriteControl sends a control frame over the underlying WebSocket.
func (s *ContextStream) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteControl(messageType, data, deadline)
}

// Close closes the underlying WebSocket.
func (s *ContextStream) Close() error {
	return s.conn.Close()
}

func (s *ContextStream) setDone(done StreamDone) {
	s.doneMu.Lock()
	defer s.doneMu.Unlock()
	if s.done != nil {
		return
	}
	copied := done
	s.done = &copied
}

func isTerminalDoneMessage(msg wsContextMessage) bool {
	if msg.Type != "done" {
		return false
	}
	return msg.RequestID == "" || msg.ExitCode != nil || msg.State != ""
}

func (s *ContextStream) normalizeInput(input StreamInput) (StreamInput, error) {
	normalized := input
	if strings.TrimSpace(normalized.Type) == "" {
		normalized.Type = "input"
	}

	switch normalized.Type {
	case "input":
		if normalized.RequestID == "" {
			normalized.RequestID = s.nextRequestID()
		}
	case "resize":
		if normalized.Rows == 0 || normalized.Cols == 0 {
			return StreamInput{}, errors.New("resize rows and cols must be > 0")
		}
	case "signal":
		if strings.TrimSpace(normalized.Signal) == "" {
			return StreamInput{}, errors.New("signal is required")
		}
	default:
		return StreamInput{}, fmt.Errorf("unsupported stream input type %q", normalized.Type)
	}

	return normalized, nil
}

func (s *ContextStream) nextRequestID() string {
	seq := atomic.AddUint64(&s.requestCounter, 1)
	return fmt.Sprintf("req-%d-%d", time.Now().UnixMilli(), seq)
}
