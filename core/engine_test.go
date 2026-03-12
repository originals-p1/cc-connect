package core

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// --- stubs for Engine tests ---

type stubAgent struct{}

func (a *stubAgent) Name() string { return "stub" }
func (a *stubAgent) StartSession(_ context.Context, _ string) (AgentSession, error) {
	return &stubAgentSession{}, nil
}
func (a *stubAgent) ListSessions(_ context.Context) ([]AgentSessionInfo, error) { return nil, nil }
func (a *stubAgent) Stop() error                                                { return nil }

type clearableStubAgent struct {
	sessions  []AgentSessionInfo
	deleted   []string
	listErr   error
	deleteErr map[string]error
}

func (a *clearableStubAgent) Name() string { return "clearable-stub" }
func (a *clearableStubAgent) StartSession(_ context.Context, _ string) (AgentSession, error) {
	return &stubAgentSession{}, nil
}
func (a *clearableStubAgent) ListSessions(_ context.Context) ([]AgentSessionInfo, error) {
	if a.listErr != nil {
		return nil, a.listErr
	}
	out := make([]AgentSessionInfo, len(a.sessions))
	copy(out, a.sessions)
	return out, nil
}
func (a *clearableStubAgent) DeleteSession(_ context.Context, sessionID string) error {
	if err := a.deleteErr[sessionID]; err != nil {
		return err
	}
	a.deleted = append(a.deleted, sessionID)
	return nil
}
func (a *clearableStubAgent) Stop() error { return nil }

type stubAgentSession struct{}

func (s *stubAgentSession) Send(_ string, _ []ImageAttachment) error             { return nil }
func (s *stubAgentSession) RespondPermission(_ string, _ PermissionResult) error { return nil }
func (s *stubAgentSession) Events() <-chan Event                                 { return make(chan Event) }
func (s *stubAgentSession) CurrentSessionID() string                             { return "stub-session" }
func (s *stubAgentSession) Alive() bool                                          { return true }
func (s *stubAgentSession) Close() error                                         { return nil }

type autoCompressAgent struct {
	session         *autoCompressSession
	compressCommand string
}

func (a *autoCompressAgent) Name() string { return "auto-compress-stub" }
func (a *autoCompressAgent) StartSession(_ context.Context, _ string) (AgentSession, error) {
	return a.session, nil
}
func (a *autoCompressAgent) ListSessions(_ context.Context) ([]AgentSessionInfo, error) {
	return nil, nil
}
func (a *autoCompressAgent) Stop() error { return nil }
func (a *autoCompressAgent) CompressCommand() string {
	return a.compressCommand
}

type autoCompressSession struct {
	events   chan Event
	sendErrs map[string]error
	sends    []string
	handler  func(prompt string) []Event
}

func newAutoCompressSession(handler func(prompt string) []Event) *autoCompressSession {
	return &autoCompressSession{
		events:   make(chan Event, 16),
		sendErrs: make(map[string]error),
		handler:  handler,
	}
}

func (s *autoCompressSession) Send(prompt string, _ []ImageAttachment) error {
	s.sends = append(s.sends, prompt)
	if err := s.sendErrs[prompt]; err != nil {
		return err
	}
	for _, evt := range s.handler(prompt) {
		s.events <- evt
	}
	return nil
}
func (s *autoCompressSession) RespondPermission(_ string, _ PermissionResult) error { return nil }
func (s *autoCompressSession) Events() <-chan Event                                 { return s.events }
func (s *autoCompressSession) CurrentSessionID() string                             { return "compress-session" }
func (s *autoCompressSession) Alive() bool                                          { return true }
func (s *autoCompressSession) Close() error                                         { return nil }

type stubPlatformEngine struct {
	n            string
	sent         []string
	buttonText   []string
	buttonLayout [][]ButtonOption
}

func (p *stubPlatformEngine) Name() string               { return p.n }
func (p *stubPlatformEngine) Start(MessageHandler) error { return nil }
func (p *stubPlatformEngine) Reply(_ context.Context, _ any, content string) error {
	p.sent = append(p.sent, content)
	return nil
}
func (p *stubPlatformEngine) Send(_ context.Context, _ any, content string) error {
	p.sent = append(p.sent, content)
	return nil
}
func (p *stubPlatformEngine) Stop() error { return nil }
func (p *stubPlatformEngine) SendWithButtons(_ context.Context, _ any, content string, buttons [][]ButtonOption) error {
	if p.n == "plain" {
		return errors.New("inline buttons not supported")
	}
	p.buttonText = append(p.buttonText, content)
	p.buttonLayout = buttons
	return nil
}

type stubInlineButtonPlatform struct {
	stubPlatformEngine
	buttonContent string
	buttonRows    [][]ButtonOption
}

func (p *stubInlineButtonPlatform) SendWithButtons(_ context.Context, _ any, content string, buttons [][]ButtonOption) error {
	p.buttonContent = content
	p.buttonRows = buttons
	return nil
}

type stubCardPlatform struct {
	stubPlatformEngine
	repliedCards []*Card
	sentCards    []*Card
	cardErr      error
}

func (p *stubCardPlatform) ReplyCard(_ context.Context, _ any, card *Card) error {
	if p.cardErr != nil {
		return p.cardErr
	}
	p.repliedCards = append(p.repliedCards, card)
	return nil
}

func (p *stubCardPlatform) SendCard(_ context.Context, _ any, card *Card) error {
	if p.cardErr != nil {
		return p.cardErr
	}
	p.sentCards = append(p.sentCards, card)
	return nil
}

type stubModelModeAgent struct {
	stubAgent
	model string
	mode  string
}

func (a *stubModelModeAgent) SetModel(model string) {
	a.model = model
}

func (a *stubModelModeAgent) GetModel() string {
	return a.model
}

func (a *stubModelModeAgent) AvailableModels(_ context.Context) []ModelOption {
	return []ModelOption{
		{Name: "gpt-4.1", Desc: "Balanced"},
		{Name: "gpt-4.1-mini", Desc: "Fast"},
	}
}

func (a *stubModelModeAgent) SetMode(mode string) {
	a.mode = mode
}

func (a *stubModelModeAgent) GetMode() string {
	if a.mode == "" {
		return "default"
	}
	return a.mode
}

func (a *stubModelModeAgent) PermissionModes() []PermissionModeInfo {
	return []PermissionModeInfo{
		{Key: "default", Name: "Default", NameZh: "默认", Desc: "Ask before risky actions", DescZh: "危险操作前询问"},
		{Key: "yolo", Name: "YOLO", NameZh: "放手做", Desc: "Skip confirmations", DescZh: "跳过确认"},
	}
}

type stubListAgent struct {
	stubAgent
	sessions []AgentSessionInfo
}

func (a *stubListAgent) ListSessions(_ context.Context) ([]AgentSessionInfo, error) {
	return a.sessions, nil
}

type stubProviderAgent struct {
	stubAgent
	providers []ProviderConfig
	active    string
}

func (a *stubProviderAgent) ListProviders() []ProviderConfig {
	return a.providers
}

func (a *stubProviderAgent) SetProviders(providers []ProviderConfig) {
	a.providers = providers
}

func (a *stubProviderAgent) GetActiveProvider() *ProviderConfig {
	for i := range a.providers {
		if a.providers[i].Name == a.active {
			return &a.providers[i]
		}
	}
	return nil
}

func (a *stubProviderAgent) SetActiveProvider(name string) bool {
	if name == "" {
		a.active = ""
		return true
	}
	for _, prov := range a.providers {
		if prov.Name == name {
			a.active = name
			return true
		}
	}
	return false
}

func newTestEngine() *Engine {
	return NewEngine("test", &stubAgent{}, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)
}

func newTestEngineWithAgent(agent Agent) *Engine {
	return NewEngine("test", agent, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)
}

func TestProcessInteractiveMessageAutoCompressesAndRetriesOnce(t *testing.T) {
	attempts := 0
	sessionStub := newAutoCompressSession(func(prompt string) []Event {
		switch prompt {
		case "hello":
			attempts++
			if attempts == 1 {
				return []Event{{Type: EventError, Error: ErrAutoCompressNeeded}}
			}
			return []Event{{Type: EventResult, Content: "retried ok", SessionID: "compress-session"}}
		case "/compress":
			return []Event{{Type: EventResult}}
		default:
			return nil
		}
	})
	agent := &autoCompressAgent{session: sessionStub, compressCommand: "/compress"}
	engine := newTestEngineWithAgent(agent)
	platform := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "s1", Content: "hello", ReplyCtx: "ctx"}
	session := engine.sessions.GetOrCreateActive(msg.SessionKey)

	engine.processInteractiveMessage(platform, msg, session)

	wantSends := []string{"hello", "/compress", "hello"}
	if !slices.Equal(sessionStub.sends, wantSends) {
		t.Fatalf("session sends = %v, want %v", sessionStub.sends, wantSends)
	}
	if len(platform.sent) == 0 || platform.sent[len(platform.sent)-1] != "retried ok" {
		t.Fatalf("platform sent = %v, want final retried response", platform.sent)
	}
}

func TestProcessInteractiveMessageStopsAfterSingleAutoCompressRetry(t *testing.T) {
	sessionStub := newAutoCompressSession(func(prompt string) []Event {
		switch prompt {
		case "hello":
			return []Event{{Type: EventError, Error: ErrAutoCompressNeeded}}
		case "/compress":
			return []Event{{Type: EventResult}}
		default:
			return []Event{{Type: EventError, Error: ErrAutoCompressNeeded}}
		}
	})
	agent := &autoCompressAgent{session: sessionStub, compressCommand: "/compress"}
	engine := newTestEngineWithAgent(agent)
	platform := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "s2", Content: "hello", ReplyCtx: "ctx"}
	session := engine.sessions.GetOrCreateActive(msg.SessionKey)

	engine.processInteractiveMessage(platform, msg, session)

	wantSends := []string{"hello", "/compress", "hello"}
	if !slices.Equal(sessionStub.sends, wantSends) {
		t.Fatalf("session sends = %v, want %v", sessionStub.sends, wantSends)
	}
	if len(platform.sent) == 0 {
		t.Fatalf("platform sent no messages")
	}
	if !strings.Contains(platform.sent[len(platform.sent)-1], "automatic compression") {
		t.Fatalf("platform final message = %q, want retry failure notice", platform.sent[len(platform.sent)-1])
	}
}

func TestProcessInteractiveEventsSendsToolResultPreview(t *testing.T) {
	sessionStub := newAutoCompressSession(func(prompt string) []Event {
		if prompt != "hello" {
			return nil
		}
		return []Event{
			{Type: EventToolUse, ToolName: "Bash", ToolInput: "pwd"},
			{Type: EventToolResult, ToolName: "Bash", ToolResult: "/tmp/project"},
			{Type: EventResult, Content: "done", SessionID: "tool-session"},
		}
	})
	engine := newTestEngineWithAgent(&autoCompressAgent{session: sessionStub, compressCommand: "/compress"})
	platform := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "tool:s1", Content: "hello", ReplyCtx: "ctx"}
	session := engine.sessions.GetOrCreateActive(msg.SessionKey)

	engine.processInteractiveMessage(platform, msg, session)

	if len(platform.sent) < 3 {
		t.Fatalf("platform sent = %v, want tool use, tool result, final reply", platform.sent)
	}
	if !strings.Contains(platform.sent[1], "Tool Result: Bash") {
		t.Fatalf("tool result message = %q, want tool result preview", platform.sent[1])
	}
	if !strings.Contains(platform.sent[1], "/tmp/project") {
		t.Fatalf("tool result message = %q, want tool output", platform.sent[1])
	}
}

type taskStubAgent struct {
	session *taskStubSession
}

func newTaskStubAgent() *taskStubAgent {
	return &taskStubAgent{
		session: &taskStubSession{
			events:     make(chan Event, 1),
			sendCalled: make(chan string, 1),
		},
	}
}

func (a *taskStubAgent) Name() string { return "task-stub" }
func (a *taskStubAgent) StartSession(_ context.Context, _ string) (AgentSession, error) {
	return a.session, nil
}
func (a *taskStubAgent) ListSessions(_ context.Context) ([]AgentSessionInfo, error) { return nil, nil }
func (a *taskStubAgent) Stop() error                                                { return nil }

type taskStubSession struct {
	events     chan Event
	sendCalled chan string
}

func (s *taskStubSession) Send(prompt string, _ []ImageAttachment) error {
	s.sendCalled <- prompt
	s.events <- Event{Type: EventResult, Content: "ok", SessionID: "task-session"}
	return nil
}
func (s *taskStubSession) RespondPermission(_ string, _ PermissionResult) error { return nil }
func (s *taskStubSession) Events() <-chan Event                                 { return s.events }
func (s *taskStubSession) CurrentSessionID() string                             { return "task-session" }
func (s *taskStubSession) Alive() bool                                          { return true }
func (s *taskStubSession) Close() error                                         { return nil }

func countCardActionValues(card *Card, prefix string) int {
	count := 0
	for _, elem := range card.Elements {
		switch e := elem.(type) {
		case CardActions:
			for _, btn := range e.Buttons {
				if strings.HasPrefix(btn.Value, prefix) {
					count++
				}
			}
		case CardListItem:
			if strings.HasPrefix(e.BtnValue, prefix) {
				count++
			}
		}
	}
	return count
}

func findCardAction(card *Card, value string) (CardButton, bool) {
	for _, elem := range card.Elements {
		switch e := elem.(type) {
		case CardActions:
			for _, btn := range e.Buttons {
				if btn.Value == value {
					return btn, true
				}
			}
		case CardListItem:
			if e.BtnValue == value {
				return CardButton{Text: e.BtnText, Type: e.BtnType, Value: e.BtnValue}, true
			}
		}
	}
	return CardButton{}, false
}

func collectCardActionRows(card *Card) []CardActions {
	rows := make([]CardActions, 0)
	for _, elem := range card.Elements {
		if row, ok := elem.(CardActions); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

// --- alias tests ---

func TestEngine_Alias(t *testing.T) {
	e := newTestEngine()
	e.AddAlias("帮助", "/help")
	e.AddAlias("新建", "/new")

	got := e.resolveAlias("帮助")
	if got != "/help" {
		t.Errorf("resolveAlias('帮助') = %q, want /help", got)
	}

	got = e.resolveAlias("新建 my-session")
	if got != "/new my-session" {
		t.Errorf("resolveAlias('新建 my-session') = %q, want '/new my-session'", got)
	}

	got = e.resolveAlias("random text")
	if got != "random text" {
		t.Errorf("resolveAlias should not modify unmatched content, got %q", got)
	}
}

func TestEngine_ClearAliases(t *testing.T) {
	e := newTestEngine()
	e.AddAlias("帮助", "/help")
	e.ClearAliases()

	got := e.resolveAlias("帮助")
	if got != "帮助" {
		t.Errorf("after ClearAliases, should not resolve, got %q", got)
	}
}

// --- banned words tests ---

func TestEngine_BannedWords(t *testing.T) {
	e := newTestEngine()
	e.SetBannedWords([]string{"spam", "BadWord"})

	if w := e.matchBannedWord("this is spam content"); w != "spam" {
		t.Errorf("expected 'spam', got %q", w)
	}
	if w := e.matchBannedWord("CONTAINS BADWORD HERE"); w != "badword" {
		t.Errorf("expected case-insensitive match 'badword', got %q", w)
	}
	if w := e.matchBannedWord("clean message"); w != "" {
		t.Errorf("expected empty, got %q", w)
	}
}

func TestEngine_BannedWordsEmpty(t *testing.T) {
	e := newTestEngine()
	if w := e.matchBannedWord("anything"); w != "" {
		t.Errorf("no banned words set, should return empty, got %q", w)
	}
}

// --- disabled commands tests ---

func TestEngine_DisabledCommands(t *testing.T) {
	e := newTestEngine()
	e.SetDisabledCommands([]string{"upgrade", "restart"})

	if !e.disabledCmds["upgrade"] {
		t.Error("upgrade should be disabled")
	}
	if !e.disabledCmds["restart"] {
		t.Error("restart should be disabled")
	}
	if e.disabledCmds["help"] {
		t.Error("help should not be disabled")
	}
}

func TestEngine_DisabledCommandsWithSlash(t *testing.T) {
	e := newTestEngine()
	e.SetDisabledCommands([]string{"/upgrade"})

	if !e.disabledCmds["upgrade"] {
		t.Error("upgrade should be disabled even when prefixed with /")
	}
}

func TestCmdClear(t *testing.T) {
	agent := &clearableStubAgent{
		sessions: []AgentSessionInfo{
			{ID: "sess-1", Summary: "first", MessageCount: 3},
			{ID: "sess-2", Summary: "second", MessageCount: 5},
			{ID: "sess-3", Summary: "third", MessageCount: 8},
		},
		deleteErr: map[string]error{},
	}
	e := newTestEngineWithAgent(agent)
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx", Content: "/clear"}

	active := e.sessions.GetOrCreateActive(msg.SessionKey)
	active.AgentSessionID = "sess-2"
	active.Name = "active"
	e.sessions.SetSessionName("sess-2", "named-active")
	e.sessions.Save()

	e.HandleIncomingMessage(p, msg)

	if len(agent.deleted) != 3 {
		t.Fatalf("deleted %d sessions, want 3", len(agent.deleted))
	}
	for i, want := range []string{"sess-1", "sess-2", "sess-3"} {
		if agent.deleted[i] != want {
			t.Fatalf("deleted[%d] = %q, want %q", i, agent.deleted[i], want)
		}
	}

	newActive := e.sessions.GetOrCreateActive(msg.SessionKey)
	if newActive.AgentSessionID != "" {
		t.Fatalf("active AgentSessionID = %q, want empty after /clear", newActive.AgentSessionID)
	}
	if got := e.sessions.GetSessionName("sess-2"); got != "" {
		t.Fatalf("session name for deleted session = %q, want empty", got)
	}
	if len(p.sent) == 0 {
		t.Fatal("expected /clear reply")
	}
	wantReply := "✅ Cleared 3 sessions."
	if p.sent[len(p.sent)-1] != wantReply {
		t.Fatalf("reply = %q, want %q", p.sent[len(p.sent)-1], wantReply)
	}
}

func TestCmdClearUnsupportedAgent(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx", Content: "/clear"}

	e.HandleIncomingMessage(p, msg)

	if len(p.sent) == 0 {
		t.Fatal("expected /clear reply")
	}
	want := "❌ This agent does not support clearing sessions."
	if p.sent[len(p.sent)-1] != want {
		t.Fatalf("reply = %q, want %q", p.sent[len(p.sent)-1], want)
	}
}

func TestCmdTaskBuildsNormalizedPrompt(t *testing.T) {
	agent := newTaskStubAgent()
	e := newTestEngineWithAgent(agent)
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx", Content: "/task 修复登录问题"}

	e.HandleIncomingMessage(p, msg)

	select {
	case prompt := <-agent.session.sendCalled:
		if prompt == msg.Content {
			t.Fatal("expected /task to normalize the prompt before sending to the agent")
		}
		for _, want := range []string{
			"Follow the project conventions and repository workflow.",
			"Complete the requested work.",
			"Avoid unnecessary questions.",
			"User requirement:",
			"修复登录问题",
		} {
			if !contains(prompt, want) {
				t.Fatalf("normalized prompt missing %q:\n%s", want, prompt)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for /task to reach the agent session")
	}
}

func TestCmdTaskRequiresBody(t *testing.T) {
	agent := newTaskStubAgent()
	e := newTestEngineWithAgent(agent)
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx", Content: "/task"}

	e.HandleIncomingMessage(p, msg)

	select {
	case prompt := <-agent.session.sendCalled:
		t.Fatalf("unexpected prompt sent to agent: %q", prompt)
	case <-time.After(100 * time.Millisecond):
	}

	if len(p.sent) == 0 {
		t.Fatal("expected /task usage reply")
	}
	want := "Usage: /task <requirement>"
	if p.sent[len(p.sent)-1] != want {
		t.Fatalf("reply = %q, want %q", p.sent[len(p.sent)-1], want)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || strings.Contains(s, sub)
}

// --- quiet tests ---

func TestQuietSessionToggle(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	// /quiet — per-session toggle on
	e.cmdQuiet(p, msg, nil)

	e.interactiveMu.Lock()
	state := e.interactiveStates["test:user1"]
	e.interactiveMu.Unlock()

	if state == nil {
		t.Fatal("expected interactiveState to be created")
	}
	state.mu.Lock()
	q := state.quiet
	state.mu.Unlock()
	if !q {
		t.Fatal("expected session quiet to be true")
	}

	// /quiet — per-session toggle off
	e.cmdQuiet(p, msg, nil)
	state.mu.Lock()
	q = state.quiet
	state.mu.Unlock()
	if q {
		t.Fatal("expected session quiet to be false after second toggle")
	}
}

func TestQuietSessionResetsOnNewSession(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	// Enable per-session quiet
	e.cmdQuiet(p, msg, nil)

	// Simulate /new
	e.cleanupInteractiveState("test:user1")

	// State should be gone, quiet resets
	e.interactiveMu.Lock()
	state := e.interactiveStates["test:user1"]
	e.interactiveMu.Unlock()
	if state != nil {
		t.Fatal("expected interactiveState to be cleaned up")
	}

	// Global quiet should still be off
	e.quietMu.RLock()
	gq := e.quiet
	e.quietMu.RUnlock()
	if gq {
		t.Fatal("expected global quiet to be false")
	}
}

func TestQuietGlobalToggle(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	// Default: global quiet is off
	if e.quiet {
		t.Fatal("expected global quiet to be false by default")
	}

	// /quiet global — toggle on
	e.cmdQuiet(p, msg, []string{"global"})
	e.quietMu.RLock()
	q := e.quiet
	e.quietMu.RUnlock()
	if !q {
		t.Fatal("expected global quiet to be true")
	}

	// /quiet global — toggle off
	e.cmdQuiet(p, msg, []string{"global"})
	e.quietMu.RLock()
	q = e.quiet
	e.quietMu.RUnlock()
	if q {
		t.Fatal("expected global quiet to be false after second toggle")
	}
}

func TestQuietGlobalPersistsAcrossSessions(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	// Enable global quiet
	e.cmdQuiet(p, msg, []string{"global"})

	// Simulate /new
	e.cleanupInteractiveState("test:user1")

	// Global quiet should still be on
	e.quietMu.RLock()
	q := e.quiet
	e.quietMu.RUnlock()
	if !q {
		t.Fatal("expected global quiet to remain true after session cleanup")
	}
}

func TestQuietGlobalAndSessionCombined(t *testing.T) {
	e := newTestEngine()
	p := &stubPlatformEngine{n: "test"}
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	// Only global quiet on — should suppress
	e.cmdQuiet(p, msg, []string{"global"})
	e.quietMu.RLock()
	gq := e.quiet
	e.quietMu.RUnlock()
	if !gq {
		t.Fatal("expected global quiet on")
	}

	// Session quiet is off (no state yet) — global alone should be enough
	e.interactiveMu.Lock()
	state := e.interactiveStates["test:user1"]
	e.interactiveMu.Unlock()
	if state != nil {
		t.Fatal("expected no session state yet")
	}

	// Turn off global, turn on session
	e.cmdQuiet(p, msg, []string{"global"}) // global off
	e.cmdQuiet(p, msg, nil)                // session on

	e.quietMu.RLock()
	gq = e.quiet
	e.quietMu.RUnlock()
	if gq {
		t.Fatal("expected global quiet off")
	}

	e.interactiveMu.Lock()
	state = e.interactiveStates["test:user1"]
	e.interactiveMu.Unlock()
	state.mu.Lock()
	sq := state.quiet
	state.mu.Unlock()
	if !sq {
		t.Fatal("expected session quiet on")
	}
}

func TestReplyWithCard_FallsBackToTextWhenPlatformHasNoCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	card := NewCard().Title("Help", "blue").Markdown("Plain fallback").Build()

	e.replyWithCard(p, "ctx", card)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if got, want := p.sent[0], card.RenderText(); got != want {
		t.Fatalf("fallback text = %q, want %q", got, want)
	}
}

func TestReplyWithCard_UsesCardSenderWhenSupported(t *testing.T) {
	p := &stubCardPlatform{stubPlatformEngine: stubPlatformEngine{n: "card"}}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	card := NewCard().Markdown("Interactive").Build()

	e.replyWithCard(p, "ctx", card)

	if len(p.repliedCards) != 1 {
		t.Fatalf("replied cards = %d, want 1", len(p.repliedCards))
	}
	if len(p.sent) != 0 {
		t.Fatalf("plain replies = %d, want 0", len(p.sent))
	}
}

func TestCmdHelp_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangChinese)
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	e.cmdHelp(p, msg)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if got := p.sent[0]; got != e.i18n.T(MsgHelp) {
		t.Fatalf("help text = %q, want legacy help text", got)
	}
	if strings.Contains(p.sent[0], "cc-connect 帮助") {
		t.Fatalf("help text = %q, should not be card title fallback", p.sent[0])
	}
}

func TestCmdList_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	sessions := []AgentSessionInfo{{ID: "session-a", Summary: "First session", MessageCount: 3, ModifiedAt: time.Date(2026, 3, 11, 2, 0, 0, 0, time.UTC)}}
	e := NewEngine("test", &stubListAgent{sessions: sessions}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	e.cmdList(p, msg, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "Sessions") {
		t.Fatalf("list text = %q, want legacy list title", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← 返回]") {
		t.Fatalf("list text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestCmdCurrent_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}
	session := e.sessions.GetOrCreateActive(msg.SessionKey)
	session.Name = "Focus"
	session.AgentSessionID = "session-123"
	session.History = append(session.History, HistoryEntry{Role: "user", Content: "hello", Timestamp: time.Now()})

	e.cmdCurrent(p, msg)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "Current session") {
		t.Fatalf("current text = %q, want legacy current session text", p.sent[0])
	}
	if strings.Contains(p.sent[0], "cc-connect") {
		t.Fatalf("current text = %q, should not be card fallback title", p.sent[0])
	}
}
func TestExecuteCardActionStop_PreservesQuietStateWithoutCleanupReinsert(t *testing.T) {
	e := newTestEngine()
	e.interactiveMu.Lock()
	e.interactiveStates["test:user1"] = &interactiveState{quiet: true}
	e.interactiveMu.Unlock()

	e.executeCardAction("/stop", "", "test:user1")

	e.interactiveMu.Lock()
	state := e.interactiveStates["test:user1"]
	e.interactiveMu.Unlock()
	if state == nil {
		t.Fatal("expected interactive state to remain for quiet preservation")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.quiet {
		t.Fatal("expected quiet state to remain enabled")
	}
	if state.pending != nil {
		t.Fatal("expected pending permission to be cleared")
	}
}

func TestCmdLang_UsesInlineButtonsOnButtonOnlyPlatform(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "inline-only"}}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)

	e.cmdLang(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.buttonRows) == 0 {
		t.Fatal("expected /lang to send inline buttons on button-only platform")
	}
	if got := p.buttonRows[0][0].Data; got != "cmd:/lang en" {
		t.Fatalf("first /lang button = %q, want %q", got, "cmd:/lang en")
	}
}

func TestCmdLang_UsesPlainTextChoicesOnPlatformWithoutCardsOrButtons(t *testing.T) {
	p := &plainBotPlatform{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)

	e.cmdLang(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "/lang en") || !strings.Contains(p.sent[0], "/lang auto") {
		t.Fatalf("lang text = %q, want plain-text language choices", p.sent[0])
	}
}

func TestCmdProvider_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	agent := &stubProviderAgent{
		providers: []ProviderConfig{
			{Name: "openai", BaseURL: "https://api.openai.com", Model: "gpt-4.1"},
			{Name: "azure", BaseURL: "https://azure.example", Model: "gpt-4.1-mini"},
		},
		active: "openai",
	}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)

	e.cmdProvider(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "Active provider") {
		t.Fatalf("provider text = %q, want current provider section", p.sent[0])
	}
	if !strings.Contains(p.sent[0], "openai") || !strings.Contains(p.sent[0], "azure") {
		t.Fatalf("provider text = %q, want provider list", p.sent[0])
	}
	if !strings.Contains(p.sent[0], "switch") {
		t.Fatalf("provider text = %q, want switch hint", p.sent[0])
	}
}

func TestCmdModel_UsesInlineButtonsOnButtonOnlyPlatform(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "inline-only"}}
	agent := &stubModelModeAgent{}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)

	e.cmdModel(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.buttonRows) == 0 {
		t.Fatal("expected /model to send inline buttons on button-only platform")
	}
	if got := p.buttonRows[0][0].Data; got != "cmd:/model 1" {
		t.Fatalf("first /model button = %q, want %q", got, "cmd:/model 1")
	}
}

func TestCmdMode_UsesInlineButtonsOnButtonOnlyPlatform(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "inline-only"}}
	agent := &stubModelModeAgent{}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)

	e.cmdMode(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.buttonRows) == 0 {
		t.Fatal("expected /mode to send inline buttons on button-only platform")
	}
	if got := p.buttonRows[0][0].Data; got != "cmd:/mode default" {
		t.Fatalf("first /mode button = %q, want %q", got, "cmd:/mode default")
	}
}

func TestCmdStatus_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}

	e.cmdStatus(p, msg)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "Status") {
		t.Fatalf("status text = %q, want legacy status text", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← Back]") {
		t.Fatalf("status text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestCmdCommands_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	e.AddCommand("deploy", "Deploy app", "ship it", "", "", "config")

	e.cmdCommands(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "/deploy") {
		t.Fatalf("commands text = %q, want legacy command list", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← Back]") {
		t.Fatalf("commands text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestCmdConfig_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)

	e.cmdConfig(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "thinking_max_len") {
		t.Fatalf("config text = %q, want legacy config list", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← Back]") {
		t.Fatalf("config text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestCmdAlias_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	e.AddAlias("ls", "/list")

	e.cmdAlias(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"}, nil)

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "ls") || !strings.Contains(p.sent[0], "/list") {
		t.Fatalf("alias text = %q, want legacy alias list", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← Back]") {
		t.Fatalf("alias text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestCmdSkills_UsesLegacyTextOnPlatformWithoutCardSupport(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	e := NewEngine("test", &stubAgent{}, []Platform{p}, "", LangEnglish)
	temp := t.TempDir()
	skillDir := temp + "/demo"
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte("---\ndescription: Demo skill\n---\nDo demo"), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
	e.skills.SetDirs([]string{temp})

	e.cmdSkills(p, &Message{SessionKey: "test:user1", ReplyCtx: "ctx"})

	if len(p.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(p.sent))
	}
	if !strings.Contains(p.sent[0], "/demo") {
		t.Fatalf("skills text = %q, want legacy skills list", p.sent[0])
	}
	if strings.Contains(p.sent[0], "[← Back]") {
		t.Fatalf("skills text = %q, should not be card fallback text", p.sent[0])
	}
}

func TestRenderListCard_MakesEveryVisibleSessionClickable(t *testing.T) {
	sessions := make([]AgentSessionInfo, 0, 7)
	base := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		sessions = append(sessions, AgentSessionInfo{
			ID:           "agent-session-" + string(rune('A'+i)),
			Summary:      "Session summary",
			MessageCount: i + 1,
			ModifiedAt:   base.Add(time.Duration(i) * time.Minute),
		})
	}

	e := NewEngine("test", &stubListAgent{sessions: sessions}, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)
	e.sessions.GetOrCreateActive("test:user1").AgentSessionID = sessions[5].ID

	card, err := e.renderListCard("test:user1", 1)
	if err != nil {
		t.Fatalf("renderListCard returned error: %v", err)
	}

	if got := countCardActionValues(card, "act:/switch "); got != len(sessions) {
		t.Fatalf("switch action count = %d, want %d", got, len(sessions))
	}

	btn, ok := findCardAction(card, "act:/switch 6")
	if !ok {
		t.Fatal("expected active session switch action to exist")
	}
	if btn.Type != "primary" {
		t.Fatalf("active session button type = %q, want primary", btn.Type)
	}
}

func TestRenderHelpCard_DefaultsToSessionTab(t *testing.T) {
	e := NewEngine("test", &stubAgent{}, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)

	card := e.renderHelpCard()
	text := card.RenderText()

	if got := countCardActionValues(card, "nav:/help "); got != 4 {
		t.Fatalf("help tab action count = %d, want 4", got)
	}
	btn, ok := findCardAction(card, "nav:/help session")
	if !ok {
		t.Fatal("expected session help tab to exist")
	}
	if btn.Type != "primary" {
		t.Fatalf("session help tab type = %q, want primary", btn.Type)
	}
	if btn.Text != "Session Management" {
		t.Fatalf("session help tab text = %q, want full title", btn.Text)
	}
	if !strings.Contains(text, "**/new**") {
		t.Fatalf("default help text = %q, want session commands", text)
	}
	if strings.Contains(text, "**Session Management**") {
		t.Fatalf("default help text = %q, should not repeat tab title in body", text)
	}
	if strings.Contains(text, "**/model**") {
		t.Fatalf("default help text = %q, should not include agent commands", text)
	}
}

func TestHandleCardNav_HelpSwitchesTabs(t *testing.T) {
	e := NewEngine("test", &stubAgent{}, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)

	card := e.handleCardNav("nav:/help agent", "test:user1")
	if card == nil {
		t.Fatal("expected help nav card")
	}
	text := card.RenderText()

	if !strings.Contains(text, "**/model**") {
		t.Fatalf("agent help text = %q, want agent commands", text)
	}
	if strings.Contains(text, "**Agent Configuration**") {
		t.Fatalf("agent help text = %q, should not repeat tab title in body", text)
	}
	if strings.Contains(text, "**/new**") {
		t.Fatalf("agent help text = %q, should not include session commands", text)
	}
}
