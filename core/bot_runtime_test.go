package core

import (
	"testing"
	"time"
)

func testCatalog() *ProjectCatalog {
	return &ProjectCatalog{
		Root: "/tmp/code",
		Projects: map[string]ProjectInfo{
			"repo-a": {Name: "repo-a", Path: "/tmp/code/repo-a", GitRoot: "/tmp/code/repo-a"},
			"repo-b": {Name: "repo-b", Path: "/tmp/code/repo-b", GitRoot: "/tmp/code/repo-b"},
			"repo-c": {Name: "repo-c", Path: "/tmp/code/repo-c", GitRoot: "/tmp/code/repo-c"},
		},
	}
}

func TestBotRuntimeManagerLazyCreateAndReuse(t *testing.T) {
	catalog := testCatalog()
	created := 0

	mgr := NewBotRuntimeManager(catalog, 2, func(botID string, proj ProjectInfo) (*ProjectRuntime, error) {
		created++
		return &ProjectRuntime{
			BotID:       botID,
			Project:     proj,
			LastUsedAt:  time.Now(),
			CloseFunc:   func() error { return nil },
			RuntimeData: created,
		}, nil
	})

	rt1, err := mgr.GetOrCreate("bot-a", "repo-a")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	rt2, err := mgr.GetOrCreate("bot-a", "repo-a")
	if err != nil {
		t.Fatalf("GetOrCreate() second call error = %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
	if rt1 != rt2 {
		t.Fatal("expected cached runtime to be reused")
	}
}

func TestBotRuntimeManagerBotIdentityIsolation(t *testing.T) {
	catalog := testCatalog()
	created := 0

	mgr := NewBotRuntimeManager(catalog, 2, func(botID string, proj ProjectInfo) (*ProjectRuntime, error) {
		created++
		return &ProjectRuntime{
			BotID:      botID,
			Project:    proj,
			LastUsedAt: time.Now(),
			CloseFunc:  func() error { return nil },
		}, nil
	})

	rt1, err := mgr.GetOrCreate("bot-a", "repo-a")
	if err != nil {
		t.Fatalf("GetOrCreate(bot-a) error = %v", err)
	}
	rt2, err := mgr.GetOrCreate("bot-b", "repo-a")
	if err != nil {
		t.Fatalf("GetOrCreate(bot-b) error = %v", err)
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}
	if rt1 == rt2 {
		t.Fatal("different bot IDs should not share runtimes")
	}
}

func TestBotRuntimeManagerEvictsLeastRecentlyUsed(t *testing.T) {
	catalog := testCatalog()
	closed := make([]string, 0, 1)

	mgr := NewBotRuntimeManager(catalog, 2, func(botID string, proj ProjectInfo) (*ProjectRuntime, error) {
		return &ProjectRuntime{
			BotID:      botID,
			Project:    proj,
			LastUsedAt: time.Now(),
			CloseFunc: func() error {
				closed = append(closed, proj.Name)
				return nil
			},
		}, nil
	})

	if _, err := mgr.GetOrCreate("bot-a", "repo-a"); err != nil {
		t.Fatalf("GetOrCreate(repo-a) error = %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := mgr.GetOrCreate("bot-a", "repo-b"); err != nil {
		t.Fatalf("GetOrCreate(repo-b) error = %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := mgr.GetOrCreate("bot-a", "repo-c"); err != nil {
		t.Fatalf("GetOrCreate(repo-c) error = %v", err)
	}

	if len(closed) != 1 || closed[0] != "repo-a" {
		t.Fatalf("closed = %v, want [repo-a]", closed)
	}
}
