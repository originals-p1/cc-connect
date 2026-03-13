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
// Each Send() launches a fresh `opencode run --format json` process and uses
// `--session` only when cc-connect already has a concrete OpenCode session ID.
type opencodeSession struct {
	cmd       string
	workDir   string
	model     string
	mode      string
	extraEnv  []string
	events    chan core.Event
	chatID    atomic.Value // stores string — OpenCode session ID
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	alive     atomic.Bool
	closeOnce sync.Once
	sendMu    sync.Mutex
	onClose   func(*opencodeSession)
}

type opencodeRunFailure struct {
	err          error
	staleSession bool
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

	fullPrompt := prompt
	if len(imageRefs) > 0 {
		fullPrompt = strings.Join(imageRefs, " ") + "\n\n" + prompt
	}

	s.wg.Add(1)
	go s.runPrompt(fullPrompt, cleanup)

	return nil
}

func (s *opencodeSession) runPrompt(fullPrompt string, cleanup func()) {
	defer s.wg.Done()
	defer cleanup()

	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	for attempt := 0; attempt < 2; attempt++ {
		failure := s.runPromptOnce(fullPrompt)
		if failure == nil {
			return
		}
		if failure.staleSession && attempt == 0 && s.CurrentSessionID() != "" {
			staleID := s.CurrentSessionID()
			slog.Warn("opencodeSession: stale session detected, retrying fresh session", "session_id", staleID)
			s.chatID.Store("")
			continue
		}
		s.emitError(failure.err)
		return
	}
}

func (s *opencodeSession) runPromptOnce(fullPrompt string) *opencodeRunFailure {
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
	args = append(args, "--thinking", fullPrompt)

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
		return &opencodeRunFailure{err: fmt.Errorf("opencodeSession: stdout pipe: %w", err)}
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return &opencodeRunFailure{err: fmt.Errorf("opencodeSession: start: %w", err)}
	}

	return s.readLoop(cmd, stdout, &stderrBuf)
}

func (s *opencodeSession) readLoop(cmd *exec.Cmd, stdout io.ReadCloser, stderrBuf *bytes.Buffer) *opencodeRunFailure {
	reader := core.NewAgentLineReader(stdout)
	var eventErr string
	var staleEventErr string

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
				return nil
			}
			slog.Error("opencodeSession: line reader error", "error", err)
			evt := core.Event{Type: core.EventError, Error: fmt.Errorf("read stdout: %w", err)}
			select {
			case s.events <- evt:
			case <-s.ctx.Done():
			}
			return nil
		}
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("opencodeSession: non-JSON line", "line", line)
			continue
		}

		if eventType, _ := raw["type"].(string); eventType == "error" {
			errMsg := opencodeErrorMessage(raw)
			if isStaleOpenCodeSessionError(errMsg) {
				staleEventErr = errMsg
				continue
			}
			eventErr = errMsg
			s.emitError(fmt.Errorf("%s", errMsg))
			continue
		}

		s.handleEvent(raw)
	}

	if err := cmd.Wait(); err != nil {
		if s.ctx.Err() != nil {
			return nil
		}
		stderrMsg := strings.TrimSpace(stderrBuf.String())
		msg := stderrMsg
		if msg == "" && staleEventErr != "" {
			msg = staleEventErr
		}
		if msg == "" && eventErr != "" {
			msg = eventErr
		}
		if msg == "" {
			msg = err.Error()
		}
		stale := isStaleOpenCodeSessionError(msg) || isStaleOpenCodeSessionError(staleEventErr)
		if eventErr != "" && !stale {
			return nil
		}
		slog.Error("opencodeSession: process failed", "error", err, "stderr", msg, "stale_session", stale)
		return &opencodeRunFailure{err: fmt.Errorf("%s", msg), staleSession: stale}
	}

	if staleEventErr != "" {
		return &opencodeRunFailure{err: fmt.Errorf("%s", staleEventErr), staleSession: true}
	}

	evt := core.Event{Type: core.EventResult, SessionID: s.CurrentSessionID(), Done: true}
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
	}

	return nil
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
	s.emitError(fmt.Errorf("%s", opencodeErrorMessage(raw)))
}

func (s *opencodeSession) handleText(raw map[string]any) {
	part, _ := raw["part"].(map[string]any)
	if part == nil {
		return
	}
	text, _ := part["text"].(string)
	if text != "" {
		if shouldSuppressOpenCodeText(text) {
			slog.Debug("opencodeSession: suppressing structured internal text payload")
			return
		}
		evt := core.Event{Type: core.EventText, Content: text}
		select {
		case s.events <- evt:
		case <-s.ctx.Done():
			return
		}
	}
}

func shouldSuppressOpenCodeText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		return false
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return false
	}

	return isInternalTaskPayload(decoded)
}

func isInternalTaskPayload(v any) bool {
	switch val := v.(type) {
	case []any:
		if len(val) == 0 {
			return false
		}
		for _, item := range val {
			obj, ok := item.(map[string]any)
			if !ok || !isTaskStateObject(obj) {
				return false
			}
		}
		return true
	case map[string]any:
		return isTaskStateObject(val)
	default:
		return false
	}
}

func isTaskStateObject(obj map[string]any) bool {
	content, ok := obj["content"].(string)
	if !ok || strings.TrimSpace(content) == "" {
		return false
	}
	status, ok := obj["status"].(string)
	if !ok || strings.TrimSpace(status) == "" {
		return false
	}
	if priority, ok := obj["priority"]; ok {
		if p, ok := priority.(string); !ok || strings.TrimSpace(p) == "" {
			return false
		}
	}
	return true
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
		return
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
}

// RespondPermission is a no-op — OpenCode handles permissions internally.
func (s *opencodeSession) RespondPermission(_ string, _ core.PermissionResult) error {
	return nil
}

func (s *opencodeSession) Events() <-chan core.Event {
	return s.events
}

func (s *opencodeSession) emitError(err error) {
	if err == nil {
		return
	}
	evt := core.Event{Type: core.EventError, Error: err}
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
	}
}

func (s *opencodeSession) CurrentSessionID() string {
	v, _ := s.chatID.Load().(string)
	return v
}

func (s *opencodeSession) Alive() bool {
	return s.alive.Load()
}

func (s *opencodeSession) Close() error {
	s.closeOnce.Do(func() {
		s.alive.Store(false)
		if s.onClose != nil {
			s.onClose(s)
		}
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
	})
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

func opencodeErrorMessage(raw map[string]any) string {
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
		return "OpenCode returned an error event"
	}
	return errMsg
}

func isStaleOpenCodeSessionError(msg string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(msg))
	if trimmed == "" {
		return false
	}
	return strings.Contains(trimmed, "session not found") ||
		strings.Contains(trimmed, "failed to list agents")
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
