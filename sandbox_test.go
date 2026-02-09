package sandbox0

import (
	"context"
	"testing"
)

func TestParseCommand(t *testing.T) {
	args, err := parseCommand(`echo "hello world"`)
	if err != nil || len(args) != 2 {
		t.Fatalf("unexpected parse result: %v %v", args, err)
	}
	if _, err := parseCommand(""); err == nil {
		t.Fatal("expected empty command error")
	}
}

func TestNormalizeStreamInput(t *testing.T) {
	msg, err := normalizeStreamInput(StreamInput{})
	if err != nil || msg.Type != StreamInputTypeInput || msg.RequestID == "" {
		t.Fatalf("unexpected normalized input: %+v err=%v", msg, err)
	}
	if _, err := normalizeStreamInput(StreamInput{Type: StreamInputTypeResize}); err == nil {
		t.Fatal("expected resize error")
	}
	if _, err := normalizeStreamInput(StreamInput{Type: StreamInputTypeSignal}); err == nil {
		t.Fatal("expected signal error")
	}
	if _, err := normalizeStreamInput(StreamInput{Type: "unknown"}); err == nil {
		t.Fatal("expected unknown type error")
	}
}

func TestGenerateRequestID(t *testing.T) {
	first := generateRequestID()
	second := generateRequestID()
	if first == "" || second == "" || first == second {
		t.Fatalf("unexpected request ids: %s %s", first, second)
	}
}

func TestSandboxRunCmdValidation(t *testing.T) {
	sandbox := &Sandbox{}
	if _, err := sandbox.Run(context.Background(), "python", ""); err == nil {
		t.Fatal("expected empty input error")
	}
	if _, err := sandbox.Cmd(context.Background(), ""); err == nil {
		t.Fatal("expected empty command error")
	}
	if _, _, _, err := sandbox.CmdStream(context.Background(), "", nil); err == nil {
		t.Fatal("expected empty command error for stream")
	}
}
