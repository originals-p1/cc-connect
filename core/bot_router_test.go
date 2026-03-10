package core

import (
	"context"
	"testing"
)

type plainBotPlatform struct {
	n    string
	sent []string
}

func (p *plainBotPlatform) Name() string                                         { return p.n }
func (p *plainBotPlatform) Start(MessageHandler) error                           { return nil }
func (p *plainBotPlatform) Reply(_ context.Context, _ any, content string) error { p.sent = append(p.sent, content); return nil }
func (p *plainBotPlatform) Send(_ context.Context, _ any, content string) error  { p.sent = append(p.sent, content); return nil }
func (p *plainBotPlatform) Stop() error                                          { return nil }

func newTestBotRouter(p *stubPlatformEngine) *BotRouter {
	catalog := testCatalog()
	runtimes := NewBotRuntimeManager(catalog, 2, func(botID string, proj ProjectInfo) (*ProjectRuntime, error) {
		return &ProjectRuntime{
			BotID:     botID,
			Project:   proj,
			Engine:    newTestEngine(),
			CloseFunc: func() error { return nil },
		}, nil
	})
	return &BotRouter{
		BotID:    "bot-a",
		DMOnly:   true,
		Catalog:  catalog,
		Bindings: NewBindingStore(""),
		Runtimes: runtimes,
	}
}

func TestBotRouterProjectList(t *testing.T) {
	p := &plainBotPlatform{n: "test"}
	router := newTestBotRouter(&stubPlatformEngine{n: "test"})

	router.HandleMessage(p, &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     true,
		Content:  "/project list",
	})

	if len(p.sent) == 0 || p.sent[0] == "" {
		t.Fatal("expected /project list reply")
	}
}

func TestBotRouterProjectListButtons(t *testing.T) {
	p := &stubPlatformEngine{n: "test"}
	router := newTestBotRouter(p)

	router.HandleMessage(p, &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     true,
		Content:  "/project-list",
	})

	if len(p.buttonText) != 1 {
		t.Fatalf("buttonText = %v, want one button response", p.buttonText)
	}
	if len(p.buttonLayout) == 0 || len(p.buttonLayout[0]) == 0 {
		t.Fatalf("buttonLayout = %v, want at least one button", p.buttonLayout)
	}
	if p.buttonLayout[0][0].Data != "cmd:/project switch repo-a" {
		t.Fatalf("first button data = %q, want project switch command", p.buttonLayout[0][0].Data)
	}
}

func TestBotRouterProjectSwitchAndCurrent(t *testing.T) {
	p := &stubPlatformEngine{n: "test"}
	router := newTestBotRouter(p)
	msg := &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     true,
		Content:  "/project switch repo-a",
	}

	router.HandleMessage(p, msg)
	router.HandleMessage(p, &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     true,
		Content:  "/project current",
	})

	if len(p.sent) < 2 {
		t.Fatalf("sent replies = %d, want at least 2", len(p.sent))
	}
	if p.sent[0] != "Switched to project repo-a." {
		t.Fatalf("switch reply = %q, want switch confirmation", p.sent[0])
	}
	if p.sent[1] != "Current project: repo-a" {
		t.Fatalf("current reply = %q, want current project", p.sent[1])
	}
}

func TestBotRouterRequiresDM(t *testing.T) {
	p := &stubPlatformEngine{n: "test"}
	router := newTestBotRouter(p)

	router.HandleMessage(p, &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     false,
		Content:  "/project list",
	})

	if len(p.sent) != 1 || p.sent[0] != "Project switching is only available in direct messages." {
		t.Fatalf("sent = %v, want DM-only error", p.sent)
	}
}

func TestBotRouterUsesDefaultProject(t *testing.T) {
	p := &stubPlatformEngine{n: "test"}
	catalog := testCatalog()
	created := 0
	router := &BotRouter{
		BotID:          "bot-a",
		DefaultProject: "repo-a",
		DMOnly:         true,
		Catalog:        catalog,
		Bindings:       NewBindingStore(""),
		Runtimes: NewBotRuntimeManager(catalog, 2, func(botID string, proj ProjectInfo) (*ProjectRuntime, error) {
			created++
			return &ProjectRuntime{
				BotID:     botID,
				Project:   proj,
				Engine:    newTestEngine(),
				CloseFunc: func() error { return nil },
			}, nil
		}),
	}

	router.HandleMessage(p, &Message{
		Platform: "telegram",
		UserID:   "u1",
		IsDM:     true,
		Content:  "hello",
	})

	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
	rec, ok := router.Bindings.Get(BindingKey{Platform: "telegram", UserID: "u1", BotID: "bot-a"})
	if !ok || rec.ActiveProject != "repo-a" {
		t.Fatalf("binding = (%+v, %v), want repo-a,true", rec, ok)
	}
	if len(p.sent) != 0 {
		t.Fatalf("sent = %v, want no router-level error reply", p.sent)
	}
}
