package copilot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

const acpProtocolVersion = 1

type copilotSession struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdinMu   sync.Mutex
	events    chan core.Event
	workDir   string
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	alive     atomic.Bool
	sessionID atomic.Value

	reqSeq    atomic.Int64
	pendingMu sync.Mutex
	pending   map[string]chan rpcEnvelope

	permMu      sync.Mutex
	permissions map[string][]permissionOption

	stateMu         sync.Mutex
	loadSupported   bool
	imageCapable    bool
	audioCapable    bool
	currentMode     string
	availableModes  []sessionMode
	configOptions   []configOption
	suppressUpdates bool
}

type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initResult struct {
	ProtocolVersion   int `json:"protocolVersion"`
	AgentCapabilities struct {
		LoadSession        bool `json:"loadSession"`
		PromptCapabilities struct {
			Image bool `json:"image"`
			Audio bool `json:"audio"`
		} `json:"promptCapabilities"`
	} `json:"agentCapabilities"`
}

type sessionSetupResult struct {
	SessionID string         `json:"sessionId"`
	Modes     *modeState     `json:"modes"`
	Config    []configOption `json:"configOptions"`
}

type modeState struct {
	CurrentModeID  string        `json:"currentModeId"`
	AvailableModes []sessionMode `json:"availableModes"`
}

type sessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type configOption struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Category     string         `json:"category"`
	Type         string         `json:"type"`
	CurrentValue any            `json:"currentValue"`
	Options      []configChoice `json:"options"`
}

type configChoice struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type permissionOption struct {
	ID   string `json:"optionId"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func newCopilotSession(ctx context.Context, cmdName, workDir, model, mode, resumeID string, extraEnv []string) (*copilotSession, error) {
	sessionCtx, cancel := context.WithCancel(ctx)

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("copilot: resolve work_dir: %w", err)
	}

	args := []string{"--acp", "--stdio"}
	cmd := exec.CommandContext(sessionCtx, cmdName, args...)
	cmd.Dir = absWorkDir
	env := os.Environ()
	if len(extraEnv) > 0 {
		env = core.MergeEnv(env, extraEnv)
	}
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("copilot: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("copilot: stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("copilot: start ACP server: %w", err)
	}

	s := &copilotSession{
		cmd:         cmd,
		stdin:       stdin,
		events:      make(chan core.Event, 64),
		workDir:     absWorkDir,
		ctx:         sessionCtx,
		cancel:      cancel,
		done:        make(chan struct{}),
		pending:     make(map[string]chan rpcEnvelope),
		permissions: make(map[string][]permissionOption),
	}
	s.alive.Store(true)

	go s.readLoop(stdout, &stderrBuf)

	if err := s.initialize(); err != nil {
		_ = s.Close()
		return nil, err
	}

	if resumeID != "" {
		if err := s.loadSession(resumeID); err != nil {
			_ = s.Close()
			return nil, err
		}
	} else {
		if err := s.createSession(); err != nil {
			_ = s.Close()
			return nil, err
		}
	}

	if err := s.applyDesiredMode(mode); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := s.applyDesiredModel(model); err != nil {
		_ = s.Close()
		return nil, err
	}

	return s, nil
}

func (s *copilotSession) initialize() error {
	var result initResult
	if err := s.request("initialize", map[string]any{
		"protocolVersion":    acpProtocolVersion,
		"clientCapabilities": map[string]any{},
		"clientInfo": map[string]any{
			"name":    "cc-connect",
			"title":   "cc-connect",
			"version": core.CurrentVersion,
		},
	}, &result); err != nil {
		return fmt.Errorf("copilot: initialize ACP: %w", err)
	}
	if result.ProtocolVersion != acpProtocolVersion {
		return fmt.Errorf("copilot: unsupported ACP version %d", result.ProtocolVersion)
	}

	s.stateMu.Lock()
	s.loadSupported = result.AgentCapabilities.LoadSession
	s.imageCapable = result.AgentCapabilities.PromptCapabilities.Image
	s.audioCapable = result.AgentCapabilities.PromptCapabilities.Audio
	s.stateMu.Unlock()
	return nil
}

func (s *copilotSession) createSession() error {
	var result sessionSetupResult
	if err := s.request("session/new", map[string]any{
		"cwd":        s.workDir,
		"mcpServers": []any{},
	}, &result); err != nil {
		return fmt.Errorf("copilot: create session: %w", err)
	}
	if result.SessionID == "" {
		return fmt.Errorf("copilot: create session returned empty sessionId")
	}
	s.sessionID.Store(result.SessionID)
	s.updateModeState(result.Modes)
	s.updateConfigOptions(result.Config)
	return nil
}

func (s *copilotSession) loadSession(sessionID string) error {
	s.stateMu.Lock()
	loadSupported := s.loadSupported
	s.suppressUpdates = true
	s.stateMu.Unlock()

	defer func() {
		s.stateMu.Lock()
		s.suppressUpdates = false
		s.stateMu.Unlock()
	}()

	if !loadSupported {
		return fmt.Errorf("copilot: ACP server does not support session/load")
	}
	if err := s.request("session/load", map[string]any{
		"sessionId":  sessionID,
		"cwd":        s.workDir,
		"mcpServers": []any{},
	}, nil); err != nil {
		return fmt.Errorf("copilot: load session %q: %w", sessionID, err)
	}
	s.sessionID.Store(sessionID)
	return nil
}

func (s *copilotSession) applyDesiredMode(mode string) error {
	mode = normalizeMode(mode)
	if mode == "" || mode == "ask" {
		return nil
	}

	s.stateMu.Lock()
	currentMode := s.currentMode
	available := append([]sessionMode(nil), s.availableModes...)
	options := append([]configOption(nil), s.configOptions...)
	s.stateMu.Unlock()

	if currentMode == mode {
		return nil
	}
	for _, m := range available {
		if m.ID == mode {
			var result struct {
				Modes *modeState `json:"modes"`
			}
			if err := s.request("session/set_mode", map[string]any{
				"sessionId": s.CurrentSessionID(),
				"modeId":    mode,
			}, &result); err != nil {
				return err
			}
			s.updateModeState(result.Modes)
			return nil
		}
	}
	if cfg := findConfigOption(options, "mode", "mode"); cfg != nil && configHasValue(*cfg, mode) {
		var result struct {
			Config []configOption `json:"configOptions"`
		}
		if err := s.request("session/set_config_option", map[string]any{
			"sessionId": s.CurrentSessionID(),
			"configId":  cfg.ID,
			"value":     mode,
		}, &result); err != nil {
			return err
		}
		s.updateConfigOptions(result.Config)
		return nil
	}

	slog.Debug("copilot: desired mode not advertised by ACP server", "mode", mode)
	return nil
}

func (s *copilotSession) applyDesiredModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}

	s.stateMu.Lock()
	options := append([]configOption(nil), s.configOptions...)
	s.stateMu.Unlock()

	cfg := findConfigOption(options, "model", "model")
	if cfg == nil || !configHasValue(*cfg, model) {
		slog.Debug("copilot: desired model not advertised by ACP server", "model", model)
		return nil
	}

	var result struct {
		Config []configOption `json:"configOptions"`
	}
	if err := s.request("session/set_config_option", map[string]any{
		"sessionId": s.CurrentSessionID(),
		"configId":  cfg.ID,
		"value":     model,
	}, &result); err != nil {
		return err
	}
	s.updateConfigOptions(result.Config)
	return nil
}

func findConfigOption(options []configOption, category, fallbackID string) *configOption {
	for i := range options {
		if options[i].Category == category || options[i].ID == fallbackID {
			return &options[i]
		}
	}
	return nil
}

func configHasValue(opt configOption, value string) bool {
	for _, choice := range opt.Options {
		if choice.Value == value {
			return true
		}
	}
	return false
}

func (s *copilotSession) Send(prompt string, images []core.ImageAttachment) error {
	if !s.alive.Load() {
		return fmt.Errorf("session process is not running")
	}

	content := make([]map[string]any, 0, len(images)+1)

	s.stateMu.Lock()
	imageCapable := s.imageCapable
	s.stateMu.Unlock()

	if prompt != "" || len(images) == 0 {
		content = append(content, map[string]any{
			"type": "text",
			"text": prompt,
		})
	}

	if len(images) > 0 {
		if !imageCapable {
			slog.Warn("copilot: ACP server does not advertise image prompt support; ignoring images")
		} else {
			for _, img := range images {
				mime := img.MimeType
				if mime == "" {
					mime = "image/png"
				}
				content = append(content, map[string]any{
					"type":     "image",
					"mimeType": mime,
					"data":     base64.StdEncoding.EncodeToString(img.Data),
				})
			}
		}
	}

	var result struct {
		StopReason string `json:"stopReason"`
	}
	if err := s.request("session/prompt", map[string]any{
		"sessionId": s.CurrentSessionID(),
		"prompt":    content,
	}, &result); err != nil {
		return fmt.Errorf("copilot: prompt: %w", err)
	}

	switch result.StopReason {
	case "", "end_turn":
		s.emit(core.Event{Type: core.EventResult, SessionID: s.CurrentSessionID(), Done: true})
	case "cancelled":
		s.emit(core.Event{Type: core.EventError, Error: context.Canceled})
	case "refusal":
		s.emit(core.Event{Type: core.EventError, Error: fmt.Errorf("copilot refused the request")})
	default:
		s.emit(core.Event{Type: core.EventResult, SessionID: s.CurrentSessionID(), Content: result.StopReason, Done: true})
	}
	return nil
}

func (s *copilotSession) RespondPermission(requestID string, result core.PermissionResult) error {
	if !s.alive.Load() {
		return fmt.Errorf("session process is not running")
	}

	s.permMu.Lock()
	options := append([]permissionOption(nil), s.permissions[requestID]...)
	delete(s.permissions, requestID)
	s.permMu.Unlock()

	if len(options) == 0 {
		return fmt.Errorf("copilot: unknown permission request %q", requestID)
	}

	var optionID string
	if result.Behavior == "allow" {
		optionID = pickPermissionOption(options, true)
	} else {
		optionID = pickPermissionOption(options, false)
	}
	if optionID == "" {
		return fmt.Errorf("copilot: no matching permission option for %q", requestID)
	}

	return s.writeJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result": map[string]any{
			"outcome": map[string]any{
				"outcome":  "selected",
				"optionId": optionID,
			},
		},
	})
}

func pickPermissionOption(options []permissionOption, allow bool) string {
	preferred := []string{}
	if allow {
		preferred = []string{"allow_once", "allow_session", "allow_always"}
	} else {
		preferred = []string{"reject_once", "reject_session", "reject_always", "deny"}
	}
	for _, kind := range preferred {
		for _, opt := range options {
			if opt.Kind == kind {
				return opt.ID
			}
		}
	}
	if len(options) == 0 {
		return ""
	}
	if allow {
		return options[0].ID
	}
	return options[len(options)-1].ID
}

func (s *copilotSession) Events() <-chan core.Event { return s.events }

func (s *copilotSession) CurrentSessionID() string {
	v, _ := s.sessionID.Load().(string)
	return v
}

func (s *copilotSession) Alive() bool { return s.alive.Load() }

func (s *copilotSession) Close() error {
	s.cancel()
	_ = s.stdin.Close()

	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("copilot: timed out closing session")
	}
	return nil
}

func (s *copilotSession) request(method string, params any, out any) error {
	id := fmt.Sprintf("%d", s.reqSeq.Add(1))
	ch := make(chan rpcEnvelope, 1)

	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()

	if err := s.writeJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return err
	}

	select {
	case msg, ok := <-ch:
		if !ok {
			return s.ctx.Err()
		}
		if msg.Error != nil {
			return fmt.Errorf("%s (code %d)", msg.Error.Message, msg.Error.Code)
		}
		if out == nil || len(msg.Result) == 0 || bytes.Equal(msg.Result, []byte("null")) {
			return nil
		}
		if err := json.Unmarshal(msg.Result, out); err != nil {
			return fmt.Errorf("decode %s response: %w", method, err)
		}
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *copilotSession) writeJSON(v any) error {
	s.stdinMu.Lock()
	defer s.stdinMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal jsonrpc: %w", err)
	}
	if _, err := s.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write jsonrpc: %w", err)
	}
	return nil
}

func (s *copilotSession) readLoop(stdout io.ReadCloser, stderrBuf *bytes.Buffer) {
	defer func() {
		s.alive.Store(false)
		s.cancelPending()
		if err := s.cmd.Wait(); err != nil {
			stderrMsg := strings.TrimSpace(stderrBuf.String())
			msg := stderrMsg
			if msg == "" {
				msg = err.Error()
			}
			s.emit(core.Event{Type: core.EventError, Error: fmt.Errorf("%s", msg)})
		}
		close(s.events)
		close(s.done)
	}()

	reader := bufio.NewReader(stdout)
	lineReader := core.NewAgentLineReader(reader)

	for {
		line, err := lineReader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			if errors.Is(err, core.ErrAgentLineSoftLimit) {
				s.emit(core.Event{Type: core.EventError, Error: core.ErrAutoCompressNeeded})
				return
			}
			s.emit(core.Event{Type: core.EventError, Error: fmt.Errorf("read copilot acp: %w", err)})
			return
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg rpcEnvelope
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			slog.Debug("copilot: non-JSON ACP line", "line", line)
			continue
		}

		switch {
		case msg.Method != "" && len(msg.ID) > 0:
			s.handleRequest(msg)
		case msg.Method != "":
			s.handleNotification(msg)
		case len(msg.ID) > 0:
			s.handleResponse(msg)
		default:
			slog.Debug("copilot: ignoring malformed ACP envelope")
		}
	}
}

func (s *copilotSession) handleResponse(msg rpcEnvelope) {
	id := normalizeRPCID(msg.ID)
	if id == "" {
		return
	}

	s.pendingMu.Lock()
	ch := s.pending[id]
	delete(s.pending, id)
	s.pendingMu.Unlock()
	if ch == nil {
		return
	}
	ch <- msg
}

func (s *copilotSession) handleRequest(msg rpcEnvelope) {
	switch msg.Method {
	case "session/request_permission":
		var params struct {
			SessionID string             `json:"sessionId"`
			ToolCall  map[string]any     `json:"toolCall"`
			Options   []permissionOption `json:"options"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			_ = s.writeJSON(map[string]any{
				"jsonrpc": "2.0",
				"id":      normalizeRPCID(msg.ID),
				"error": map[string]any{
					"code":    -32602,
					"message": "invalid params",
				},
			})
			return
		}

		requestID := normalizeRPCID(msg.ID)
		s.permMu.Lock()
		s.permissions[requestID] = append([]permissionOption(nil), params.Options...)
		s.permMu.Unlock()

		title, _ := params.ToolCall["title"].(string)
		if title == "" {
			title, _ = params.ToolCall["kind"].(string)
		}
		toolInput := summarizeToolCall(params.ToolCall)
		if toolInput == "" {
			toolInput = title
		}
		s.emit(core.Event{
			Type:      core.EventPermissionRequest,
			RequestID: requestID,
			ToolName:  title,
			ToolInput: toolInput,
		})
	default:
		_ = s.writeJSON(map[string]any{
			"jsonrpc": "2.0",
			"id":      normalizeRPCID(msg.ID),
			"error": map[string]any{
				"code":    -32601,
				"message": "method not found",
			},
		})
	}
}

func (s *copilotSession) handleNotification(msg rpcEnvelope) {
	if msg.Method != "session/update" {
		return
	}

	s.stateMu.Lock()
	suppress := s.suppressUpdates
	s.stateMu.Unlock()
	if suppress {
		return
	}

	var params struct {
		SessionID string          `json:"sessionId"`
		Update    json.RawMessage `json:"update"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return
	}

	var kind struct {
		SessionUpdate string `json:"sessionUpdate"`
	}
	if err := json.Unmarshal(params.Update, &kind); err != nil {
		return
	}

	switch kind.SessionUpdate {
	case "agent_message_chunk":
		s.handleAgentMessageChunk(params.Update)
	case "tool_call", "tool_call_update":
		s.handleToolCallUpdate(params.Update)
	case "plan":
		s.handlePlanUpdate(params.Update)
	case "current_mode_update":
		s.handleModeUpdate(params.Update)
	case "config_option_update":
		s.handleConfigUpdate(params.Update)
	}
}

func (s *copilotSession) handleAgentMessageChunk(raw json.RawMessage) {
	var update struct {
		Content map[string]any `json:"content"`
	}
	if err := json.Unmarshal(raw, &update); err != nil {
		return
	}
	kind, _ := update.Content["type"].(string)
	if kind != "text" {
		return
	}
	text, _ := update.Content["text"].(string)
	if text == "" {
		return
	}
	s.emit(core.Event{Type: core.EventText, Content: text})
}

func (s *copilotSession) handleToolCallUpdate(raw json.RawMessage) {
	var update map[string]any
	if err := json.Unmarshal(raw, &update); err != nil {
		return
	}
	status, _ := update["status"].(string)
	if status == "completed" || status == "cancelled" {
		return
	}
	title, _ := update["title"].(string)
	kind, _ := update["kind"].(string)
	if title == "" {
		title = kind
	}
	s.emit(core.Event{
		Type:      core.EventToolUse,
		ToolName:  title,
		ToolInput: summarizeToolCall(update),
	})
}

func (s *copilotSession) handlePlanUpdate(raw json.RawMessage) {
	var update struct {
		Entries []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &update); err != nil {
		return
	}
	var parts []string
	for _, entry := range update.Entries {
		if entry.Content == "" {
			continue
		}
		if entry.Status != "" {
			parts = append(parts, entry.Status+": "+entry.Content)
		} else {
			parts = append(parts, entry.Content)
		}
	}
	if len(parts) == 0 {
		return
	}
	s.emit(core.Event{Type: core.EventThinking, Content: strings.Join(parts, "\n")})
}

func (s *copilotSession) handleModeUpdate(raw json.RawMessage) {
	var update struct {
		CurrentModeID string `json:"currentModeId"`
		ModeID        string `json:"modeId"`
	}
	if err := json.Unmarshal(raw, &update); err != nil {
		return
	}
	modeID := update.CurrentModeID
	if modeID == "" {
		modeID = update.ModeID
	}
	if modeID == "" {
		return
	}
	s.stateMu.Lock()
	s.currentMode = modeID
	s.stateMu.Unlock()
}

func (s *copilotSession) handleConfigUpdate(raw json.RawMessage) {
	var update struct {
		Config []configOption `json:"configOptions"`
	}
	if err := json.Unmarshal(raw, &update); err != nil {
		return
	}
	s.updateConfigOptions(update.Config)
}

func (s *copilotSession) updateModeState(state *modeState) {
	if state == nil {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.currentMode = state.CurrentModeID
	s.availableModes = append([]sessionMode(nil), state.AvailableModes...)
}

func (s *copilotSession) updateConfigOptions(options []configOption) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.configOptions = append([]configOption(nil), options...)
}

func (s *copilotSession) emit(evt core.Event) {
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
	}
}

func (s *copilotSession) cancelPending() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for id, ch := range s.pending {
		delete(s.pending, id)
		close(ch)
	}
}

func normalizeRPCID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return strings.TrimSpace(string(raw))
}

func summarizeToolCall(raw map[string]any) string {
	var parts []string
	if title, _ := raw["title"].(string); title != "" {
		parts = append(parts, title)
	}
	if kind, _ := raw["kind"].(string); kind != "" {
		parts = append(parts, "kind="+kind)
	}
	if status, _ := raw["status"].(string); status != "" {
		parts = append(parts, "status="+status)
	}
	if id, _ := raw["toolCallId"].(string); id != "" {
		parts = append(parts, "id="+id)
	}
	return strings.Join(parts, " ")
}
