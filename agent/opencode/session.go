package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/chenhg5/cc-connect/core"
)

// opencodeSession manages multi-turn conversations with the OpenCode CLI.
// Each Send() launches a new `opencode run --format json` process
// with --session for conversation continuity.
type opencodeSession struct {
	cmd      string
	workDir  string
	model    string
	mode     string
	extraEnv []string
	events   chan core.Event
	chatID   atomic.Value // stores string — OpenCode session ID
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	alive    atomic.Bool
}

func newOpencodeSession(ctx context.Context, cmd, workDir, model, mode, resumeID string, extraEnv []string) (*opencodeSession, error) {
	sessionCtx, cancel := context.WithCancel(ctx)

	s := &opencodeSession{
		cmd:      cmd,
		workDir:  workDir,
		model:    model,
		mode:     mode,
		extraEnv: extraEnv,
		events:   make(chan core.Event, 64),
		ctx:      sessionCtx,
		cancel:   cancel,
	}
	s.alive.Store(true)

	if resumeID != "" {
		s.chatID.Store(resumeID)
	}

	return s, nil
}

func (s *opencodeSession) Send(prompt string, images []core.ImageAttachment) error {
	if !s.alive.Load() {
		return fmt.Errorf("session is closed")
	}

	imageRefs, cleanup, err := writeImageRefs(images)
	if err != nil {
		return err
	}

	chatID := s.CurrentSessionID()
	isResume := chatID != ""

	args := []string{"run", "--format", "json"}

	if isResume {
		args = append(args, "--session", chatID)
	}
	if s.model != "" {
		args = append(args, "--model", s.model)
	}
	if s.workDir != "" {
		args = append(args, "--dir", s.workDir)
	}
	if s.mode == "yolo" {
		args = append(args, "--agent", "coder")
	}

	// Enable thinking blocks
	args = append(args, "--thinking")

	fullPrompt := prompt
	if len(imageRefs) > 0 {
		fullPrompt = strings.Join(imageRefs, " ") + "\n\n" + prompt
	}

	// Append prompt as positional arg
	args = append(args, fullPrompt)

	slog.Debug("opencodeSession: launching", "resume", isResume, "args", core.RedactArgs(args))

	cmd := exec.CommandContext(s.ctx, s.cmd, args...)
	cmd.Dir = s.workDir
	env := os.Environ()
	if len(s.extraEnv) > 0 {
		env = core.MergeEnv(env, s.extraEnv)
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("opencodeSession: stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		cleanup()
		return fmt.Errorf("opencodeSession: start: %w", err)
	}

	s.wg.Add(1)
	go s.readLoop(cmd, stdout, &stderrBuf, cleanup)

	return nil
}

func (s *opencodeSession) readLoop(cmd *exec.Cmd, stdout io.ReadCloser, stderrBuf *bytes.Buffer, cleanup func()) {
	defer s.wg.Done()
	defer func() {
		cleanup()
		if err := cmd.Wait(); err != nil {
			stderrMsg := strings.TrimSpace(stderrBuf.String())
			if stderrMsg == "" {
				stderrMsg = err.Error()
			}
			slog.Error("opencodeSession: process failed", "error", err, "stderr", stderrMsg)
			evt := core.Event{Type: core.EventError, Error: fmt.Errorf("%s", stderrMsg)}
			select {
			case s.events <- evt:
			case <-s.ctx.Done():
				return
			}
		}
	}()

	reader := core.NewAgentLineReader(stdout)

	for {
		line, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, core.ErrAgentLineSoftLimit) {
				evt := core.Event{Type: core.EventError, Error: core.ErrAutoCompressNeeded}
				select {
				case s.events <- evt:
				case <-s.ctx.Done():
				}
				return
			}
			slog.Error("opencodeSession: line reader error", "error", err)
			evt := core.Event{Type: core.EventError, Error: fmt.Errorf("read stdout: %w", err)}
			select {
			case s.events <- evt:
			case <-s.ctx.Done():
			}
			return
		}
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("opencodeSession: non-JSON line", "line", line)
			continue
		}

		s.handleEvent(raw)
	}
}

// OpenCode NDJSON event structure:
//
//	{ "type": "text|tool_use|reasoning|step_start|step_finish",
//	  "part": { "type": "text|tool|reasoning|step-start|step-finish", ... } }
func (s *opencodeSession) handleEvent(raw map[string]any) {
	eventType, _ := raw["type"].(string)

	switch eventType {
	case "error":
		s.handleError(raw)
	case "text":
		s.handleText(raw)
	case "tool_use":
		s.handleToolUse(raw)
	case "reasoning":
		s.handleReasoning(raw)
	case "step_start":
		s.handleStepStart(raw)
	case "step_finish":
		s.handleStepFinish(raw)
	default:
		slog.Debug("opencodeSession: unhandled event", "type", eventType)
	}
}

func (s *opencodeSession) handleError(raw map[string]any) {
	errMsg := ""
	if errObj, ok := raw["error"].(map[string]any); ok {
		if data, ok := errObj["data"].(map[string]any); ok {
			errMsg, _ = data["message"].(string)
		}
		if errMsg == "" {
			errMsg, _ = errObj["message"].(string)
		}
	}
	if errMsg == "" {
		errMsg, _ = raw["message"].(string)
	}
	if strings.TrimSpace(errMsg) == "" {
		errMsg = "OpenCode returned an error event"
	}
	evt := core.Event{Type: core.EventError, Error: fmt.Errorf("%s", errMsg)}
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
	}
}

func (s *opencodeSession) handleText(raw map[string]any) {
	part, _ := raw["part"].(map[string]any)
	if part == nil {
		return
	}
	text, _ := part["text"].(string)
	if text != "" {
		evt := core.Event{Type: core.EventText, Content: text}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *opencodeSession) handleToolUse(raw map[string]any) {
	part, _ := raw["part"].(map[string]any)
	if part == nil {
		return
	}

	toolName, _ := part["tool"].(string)

	// Check state for status
	state, _ := part["state"].(map[string]any)
	status := ""
	if state != nil {
		status, _ = state["status"].(string)
	}

	if status == "completed" {
		// Tool result
		output := lookupString(state, "output", "result", "content")
		if state != nil {
			if output == "" {
				if outputMap, ok := state["output"].(map[string]any); ok {
					b, _ := json.Marshal(outputMap)
					output = string(b)
				}
			}
		}
		evt := core.Event{Type: core.EventToolResult, ToolName: toolName, ToolResult: truncate(output, 500)}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	} else {
		// Tool use (running or starting)
		input := ""
		if state != nil {
			inputVal, _ := state["input"].(string)
			if inputVal != "" {
				input = inputVal
			} else {
				// Try to marshal input as JSON summary
				if inputMap, ok := state["input"].(map[string]any); ok {
					b, _ := json.Marshal(inputMap)
					input = truncate(string(b), 200)
				}
			}
		}
		evt := core.Event{Type: core.EventToolUse, ToolName: toolName, ToolInput: input}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *opencodeSession) handleReasoning(raw map[string]any) {
	part, _ := raw["part"].(map[string]any)
	if part == nil {
		return
	}
	text, _ := part["text"].(string)
	if text != "" {
		evt := core.Event{Type: core.EventThinking, Content: text}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *opencodeSession) handleStepStart(raw map[string]any) {
	part, _ := raw["part"].(map[string]any)
	if part == nil {
		return
	}
	sessionID, _ := part["sessionID"].(string)
	if sessionID != "" {
		s.chatID.Store(sessionID)
		slog.Debug("opencodeSession: session started", "session_id", sessionID)
	}
}

func (s *opencodeSession) handleStepFinish(_ map[string]any) {
	sid := s.CurrentSessionID()
	evt := core.Event{Type: core.EventResult, SessionID: sid, Done: true}
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
		return
	}
}

// RespondPermission is a no-op — OpenCode handles permissions internally.
func (s *opencodeSession) RespondPermission(_ string, _ core.PermissionResult) error {
	return nil
}

func (s *opencodeSession) Events() <-chan core.Event {
	return s.events
}

func (s *opencodeSession) CurrentSessionID() string {
	v, _ := s.chatID.Load().(string)
	return v
}

func (s *opencodeSession) Alive() bool {
	return s.alive.Load()
}

func (s *opencodeSession) Close() error {
	s.alive.Store(false)
	s.cancel()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		slog.Warn("opencodeSession: close timed out, abandoning wg.Wait")
	}
	close(s.events)
	return nil
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "..."
}

func lookupString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func writeImageRefs(images []core.ImageAttachment) ([]string, func(), error) {
	if len(images) == 0 {
		return nil, func() {}, nil
	}
	tmpDir, err := os.MkdirTemp("", "cc-connect-opencode-")
	if err != nil {
		return nil, nil, fmt.Errorf("opencodeSession: create temp dir: %w", err)
	}
	refs := make([]string, 0, len(images))
	cleanup := func() { _ = os.RemoveAll(tmpDir) }
	for i, img := range images {
		ext := fileExtForImage(img)
		path := filepath.Join(tmpDir, fmt.Sprintf("image-%d%s", i, ext))
		if err := os.WriteFile(path, img.Data, 0o600); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("opencodeSession: save image: %w", err)
		}
		refs = append(refs, path)
	}
	return refs, cleanup, nil
}

func fileExtForImage(img core.ImageAttachment) string {
	if img.FileName != "" {
		if ext := filepath.Ext(img.FileName); ext != "" {
			return ext
		}
	}
	if exts, _ := mime.ExtensionsByType(img.MimeType); len(exts) > 0 {
		return exts[0]
	}
	switch img.MimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}
