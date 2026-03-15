package copilot

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func writeFakeCopilot(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake copilot is not supported on windows")
	}

	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "copilot")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"1","result":{"protocolVersion":1,"agentCapabilities":{"loadSession":true,"promptCapabilities":{"image":true}}}}'
      ;;
    *'"method":"session/new"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"2","result":{"sessionId":"sess-new","modes":{"currentModeId":"ask","availableModes":[{"id":"ask","name":"Ask"},{"id":"plan","name":"Plan"}]},"configOptions":[{"id":"model","name":"Model","category":"model","type":"select","currentValue":"gpt-4.1","options":[{"value":"gpt-4.1","name":"GPT-4.1"},{"value":"gpt-5","name":"GPT-5"}]}]}}'
      ;;
    *'"method":"session/load"'*)
      printf '%s\n' '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess-loaded","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"history should stay hidden"}}}}'
      printf '%s\n' '{"jsonrpc":"2.0","id":"2","result":null}'
      ;;
    *'"method":"session/set_mode"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"3","result":{"modes":{"currentModeId":"plan","availableModes":[{"id":"ask","name":"Ask"},{"id":"plan","name":"Plan"}]}}}'
      ;;
    *'"method":"session/set_config_option"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"4","result":{"configOptions":[{"id":"model","name":"Model","category":"model","type":"select","currentValue":"gpt-5","options":[{"value":"gpt-4.1","name":"GPT-4.1"},{"value":"gpt-5","name":"GPT-5"}]}]}}'
      ;;
    *'"method":"session/prompt"'*)
      printf '%s\n' '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess-new","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"stream chunk"}}}}'
      printf '%s\n' '{"jsonrpc":"2.0","id":"99","method":"session/request_permission","params":{"sessionId":"sess-new","toolCall":{"toolCallId":"tool-1","title":"Run command","kind":"execute","status":"pending"},"options":[{"optionId":"allow-once","name":"Allow once","kind":"allow_once"},{"optionId":"reject-once","name":"Reject","kind":"reject_once"}]}}'
      IFS= read -r reply
      case "$reply" in
        *'"optionId":"allow-once"'*)
          printf '%s\n' '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess-new","update":{"sessionUpdate":"tool_call_update","toolCallId":"tool-1","title":"Run command","kind":"execute","status":"in_progress"}}}'
          printf '%s\n' '{"jsonrpc":"2.0","id":"5","result":{"stopReason":"end_turn"}}'
          ;;
        *)
          printf '%s\n' '{"jsonrpc":"2.0","id":"5","result":{"stopReason":"refusal"}}'
          ;;
      esac
      ;;
  esac
done
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake copilot: %v", err)
	}

	pathEnv := binDir
	if current := os.Getenv("PATH"); current != "" {
		pathEnv += string(os.PathListSeparator) + current
	}
	t.Setenv("PATH", pathEnv)
}

func collectEvents(t *testing.T, ch <-chan core.Event, wait time.Duration, limit int) []core.Event {
	t.Helper()
	var events []core.Event
	timer := time.NewTimer(wait)
	defer timer.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, evt)
			if limit > 0 && len(events) >= limit {
				return events
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(wait)
		case <-timer.C:
			return events
		}
	}
}

func TestCopilotSessionPromptFlow(t *testing.T) {
	writeFakeCopilot(t)

	s, err := newCopilotSession(context.Background(), "copilot", t.TempDir(), "gpt-5", "plan", "", nil)
	if err != nil {
		t.Fatalf("newCopilotSession() error = %v", err)
	}
	defer s.Close()

	if got := s.CurrentSessionID(); got != "sess-new" {
		t.Fatalf("CurrentSessionID() = %q, want %q", got, "sess-new")
	}

	var (
		mu     sync.Mutex
		events []core.Event
		done   = make(chan struct{})
		errCh  = make(chan error, 1)
	)
	go func() {
		defer close(done)
		for evt := range s.Events() {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
			if evt.Type == core.EventPermissionRequest {
				if err := s.RespondPermission(evt.RequestID, core.PermissionResult{Behavior: "allow"}); err != nil {
					errCh <- err
					return
				}
			}
			if evt.Type == core.EventResult {
				return
			}
		}
	}()

	if err := s.Send("hello", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("RespondPermission() error = %v", err)
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event collector")
	}

	mu.Lock()
	gotEvents := append([]core.Event(nil), events...)
	mu.Unlock()

	if len(gotEvents) != 4 {
		t.Fatalf("events len = %d, want 4, events=%+v", len(gotEvents), gotEvents)
	}
	if gotEvents[0].Type != core.EventText || gotEvents[0].Content != "stream chunk" {
		t.Fatalf("events[0] = %+v, want streamed text", gotEvents[0])
	}
	if gotEvents[1].Type != core.EventPermissionRequest {
		t.Fatalf("events[1] = %+v, want permission request", gotEvents[1])
	}
	if gotEvents[2].Type != core.EventToolUse || !strings.Contains(gotEvents[2].ToolInput, "status=in_progress") {
		t.Fatalf("events[2] = %+v, want tool_use in_progress", gotEvents[2])
	}
	if gotEvents[3].Type != core.EventResult || !gotEvents[3].Done {
		t.Fatalf("events[3] = %+v, want final result", gotEvents[3])
	}
}

func TestCopilotSessionLoadSuppressesReplay(t *testing.T) {
	writeFakeCopilot(t)

	s, err := newCopilotSession(context.Background(), "copilot", t.TempDir(), "", "", "sess-loaded", nil)
	if err != nil {
		t.Fatalf("newCopilotSession(load) error = %v", err)
	}
	defer s.Close()

	events := collectEvents(t, s.Events(), 100*time.Millisecond, 1)
	if len(events) != 0 {
		t.Fatalf("events = %+v, want no replay events during session/load", events)
	}
	if got := s.CurrentSessionID(); got != "sess-loaded" {
		t.Fatalf("CurrentSessionID() = %q, want %q", got, "sess-loaded")
	}
}
