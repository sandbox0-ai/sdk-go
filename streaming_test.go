package sandbox0

import (
	"errors"
	"net"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNormalizeStreamInput(t *testing.T) {
	input, err := normalizeStreamInput(StreamInput{Data: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Type != StreamInputTypeInput || input.RequestID == "" {
		t.Fatalf("expected default input type and request id, got %#v", input)
	}

	if _, err := normalizeStreamInput(StreamInput{Type: StreamInputTypeResize}); err == nil {
		t.Fatal("expected error for resize without rows/cols")
	}
	if _, err := normalizeStreamInput(StreamInput{Type: StreamInputTypeSignal, Signal: " "}); err == nil {
		t.Fatal("expected error for missing signal")
	}
	if _, err := normalizeStreamInput(StreamInput{Type: "noop"}); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestIsStreamClosed(t *testing.T) {
	done := make(chan struct{})
	if isStreamClosed(net.ErrClosed, done) != true {
		t.Fatal("expected net.ErrClosed to be treated as closed")
	}
	close(done)
	if isStreamClosed(errors.New("any"), done) != true {
		t.Fatal("expected done channel to mark closed")
	}

	if isStreamClosed(&websocket.CloseError{Code: websocket.CloseNormalClosure}, make(chan struct{})) != true {
		t.Fatal("expected CloseNormalClosure to be treated as closed")
	}
}
